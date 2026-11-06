package _interface

type Scanner interface {
	Scan(sbom []byte, vexDocuments [][]byte) ([]byte, error)
}
