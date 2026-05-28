package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anchore/clio"
	"github.com/anchore/grype/grype"
	v6dist "github.com/anchore/grype/grype/db/v6/distribution"
	v6inst "github.com/anchore/grype/grype/db/v6/installation"
	"github.com/anchore/grype/grype/distro"
	"github.com/anchore/grype/grype/grypeerr"
	"github.com/anchore/grype/grype/matcher"
	"github.com/anchore/grype/grype/matcher/dotnet"
	"github.com/anchore/grype/grype/matcher/golang"
	"github.com/anchore/grype/grype/matcher/java"
	"github.com/anchore/grype/grype/matcher/javascript"
	"github.com/anchore/grype/grype/matcher/python"
	"github.com/anchore/grype/grype/matcher/ruby"
	"github.com/anchore/grype/grype/matcher/stock"
	grype_pkg "github.com/anchore/grype/grype/pkg"
	"github.com/anchore/grype/grype/presenter/models"
	"github.com/anchore/grype/grype/vulnerability"
	"github.com/anchore/syft/syft/format"
	log "github.com/sirupsen/logrus"

	"github.com/openclarity/grype-server/grype-server/pkg/rest"
)

const (
	defaultMavenBaseURL = "https://search.maven.org/solrsearch/select"
)

type Config struct {
	RestServerPort int
	DbRootDir      string
	DbUpdateURL    string
}

type Scanner struct {
	restServer    *rest.Server
	DbRootDir     string
	DbUpdateURL   string
	distConfig    v6dist.Config
	installConfig v6inst.Config

	sync.RWMutex
	vulProvider    vulnerability.Provider
	providerStatus *vulnerability.ProviderStatus
}

func Create(conf *Config) (*Scanner, error) {
	s := &Scanner{
		DbRootDir:   conf.DbRootDir,
		DbUpdateURL: conf.DbUpdateURL,
		distConfig: func() v6dist.Config {
			cfg := v6dist.DefaultConfig()
			cfg.ID = clio.Identification{Name: "grype-server"}
			if conf.DbUpdateURL != "" {
				cfg.LatestURL = conf.DbUpdateURL
			}
			return cfg
		}(),
		installConfig: func() v6inst.Config {
			cfg := v6inst.DefaultConfig(clio.Identification{Name: "grype-server"})
			cfg.DBRootDir = conf.DbRootDir
			cfg.ValidateAge = false
			return cfg
		}(),
	}

	var err error
	s.restServer, err = rest.CreateRESTServer(conf.RestServerPort, s)
	if err != nil {
		return nil, fmt.Errorf("failed to start rest server: %v", err)
	}

	return s, nil
}

func (s *Scanner) Start(ctx context.Context, errChan chan struct{}) error {
	if err := s.loadDB(); err != nil {
		return fmt.Errorf("failed to load DB: %v", err)
	}
	s.startUpdateChecker(ctx)
	s.restServer.Start(errChan)

	return nil
}

func (s *Scanner) Stop() {
	if s.restServer != nil {
		s.restServer.Stop()
	}
}

func (s *Scanner) Scan(sbom []byte) ([]byte, error) {
	log.Tracef("SBOM to scan: %s", sbom)
	doc, err := s.ScanSbomJson(string(sbom))
	if err != nil {
		return nil, fmt.Errorf("failed to scan SBOM: %v", err)
	}

	docB, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scan result: %v", err)
	}
	log.Tracef("Scan result: %s", docB)

	return docB, nil
}

func (s *Scanner) ScanSbomJson(sbom string) (*models.Document, error) {
	s.RLock()
	provider := s.vulProvider
	s.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("vulnerability provider wasn't set")
	}

	sbomReader := strings.NewReader(sbom)
	syftSbom, _, _, err := format.Decode(sbomReader)
	if err != nil {
		return nil, fmt.Errorf("unable to decode sbom: %v", err)
	}

	if syftSbom.Artifacts.Packages == nil {
		return nil, fmt.Errorf("packagecatalog is empty")
	}

	pkgPtrs := grype_pkg.FromCollection(syftSbom.Artifacts.Packages, syftSbom.Relationships, grype_pkg.SynthesisConfig{
		GenerateMissingCPEs: true,
	})
	packagesContext := grype_pkg.Context{
		Source: &syftSbom.Source,
		Distro: distro.FromRelease(syftSbom.Artifacts.LinuxDistribution, nil),
	}
	// Propagate distro to each package so distro-aware matchers (e.g. APK SecDB filtering) work correctly.
	// The grype CLI does this in pkg.Provide; we bypass that path so we must do it here.
	if packagesContext.Distro != nil {
		for _, p := range pkgPtrs {
			if p.Distro == nil {
				p.Distro = packagesContext.Distro
			}
		}
	}
	packages := make([]grype_pkg.Package, len(pkgPtrs))
	for i, p := range pkgPtrs {
		packages[i] = *p
	}

	doc, err := s.scanWithRetries(packagesContext, packages)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Scanner) scan(packagesContext grype_pkg.Context, packages []grype_pkg.Package) (*models.Document, error) {
	s.RLock()
	provider := s.vulProvider
	s.RUnlock()

	vulnerabilityMatcher := createVulnerabilityMatcher(provider)

	allMatches, ignoredMatches, err := vulnerabilityMatcher.FindMatches(packages, packagesContext)
	if err != nil && !errors.Is(err, grypeerr.ErrAboveSeverityThreshold) {
		return nil, fmt.Errorf("failed to find vulnerabilities: %v", err)
	}

	doc, err := models.NewDocument(clio.Identification{}, packages, packagesContext, *allMatches, ignoredMatches, provider, nil, nil, models.SortByPackage, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %v", err)
	}

	return &doc, nil
}

func createVulnerabilityMatcher(provider vulnerability.Provider) *grype.VulnerabilityMatcher {
	matchers := matcher.NewDefaultMatchers(matcher.Config{
		Java: java.MatcherConfig{
			ExternalSearchConfig: java.ExternalSearchConfig{
				SearchMavenUpstream: false,
				MavenBaseURL:        defaultMavenBaseURL,
			},
			UseCPEs: true,
		},
		Ruby:       ruby.MatcherConfig{UseCPEs: true},
		Python:     python.MatcherConfig{UseCPEs: true},
		Dotnet:     dotnet.MatcherConfig{UseCPEs: true},
		Javascript: javascript.MatcherConfig{UseCPEs: true},
		Golang:     golang.MatcherConfig{UseCPEs: true},
		Stock:      stock.MatcherConfig{UseCPEs: true},
	})

	return &grype.VulnerabilityMatcher{
		VulnerabilityProvider: provider,
		Matchers:              matchers,
		NormalizeByCVE:        true,
	}
}

const (
	numOfScanAttempts = 5
	scanRetryInterval = 5 * time.Second
)

func (s *Scanner) scanWithRetries(packagesContext grype_pkg.Context, packages []grype_pkg.Package) (*models.Document, error) {
	var err error
	var ret *models.Document

	for attempt := 1; attempt <= numOfScanAttempts; attempt++ {
		ret, err = s.scan(packagesContext, packages)
		if err != nil {
			log.Errorf("Failed to scan (attempt %v): %v", attempt, err)
			time.Sleep(scanRetryInterval)
		} else {
			return ret, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan after %v attempts: %v", numOfScanAttempts, err)
	}

	return ret, nil
}

func (s *Scanner) startUpdateChecker(ctx context.Context) {
	const checkForUpdatesIntervalSec = 3 * 60 * 60

	go func() {
		checkInterval := checkForUpdatesIntervalSec * time.Second
		for {
			select {
			case <-ctx.Done():
				log.Debugf("Stopping DB update checker")
				return
			case <-time.After(checkInterval):
				if err := s.loadDB(); err != nil {
					log.Errorf("Failed to update DB: %v", err)
				}
			}
		}
	}()
}

func (s *Scanner) loadDB() error {
	provider, status, err := grype.LoadVulnerabilityDB(s.distConfig, s.installConfig, true)
	if err != nil {
		return fmt.Errorf("failed to load vulnerability DB: %w", err)
	}

	s.Lock()
	s.vulProvider = provider
	s.providerStatus = status
	s.Unlock()

	log.Infof("Vulnerability DB loaded (built: %s, schema: %s)", status.Built.UTC().Format(time.RFC3339), status.SchemaVersion)
	return nil
}
