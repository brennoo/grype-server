# grype-server

SBOM scanning using [grype](https://github.com/anchore/grype) wrapped in a
convenient REST API.

This allows for a centralised install of grype which will sync the
vulnerability DB periodically instead of requiring all clients to have access
to the internet and the bandwidth to download the vulnerability DB.

## Table of Contents<!-- omit in toc -->

- [Usage](#usage)
- [Contributing](#contributing)
- [Code of Conduct](#code-of-conduct)
- [License](#license)

## Usage

### Running

```
docker run -d -p 9991:9991 --name grype-server <registry-name>/grype-server run --log-level info
```

### Scanning an SBOM

```
curl -X POST http://<ip>:9991/api/scanSBOM --data-binary @- <<'EOF'
{
    "sbom": "<base 64 encoded SBOM>"
}
EOF
```

### Scanning an SBOM with VEX documents

VEX documents let you suppress findings that have already been triaged as
not affecting your product. Both [OpenVEX](https://openvex.dev/) and
[CSAF](https://oasis-open.github.io/csaf-documentation/) are supported and
auto-detected — every document in `vexDocuments` must be the same format,
since detection is based on the first entry.

```
curl -X POST http://<ip>:9991/api/scanSBOM --data-binary @- <<EOF
{
    "sbom": "$(base64 < sbom.json | tr -d '\n')",
    "vexDocuments": ["$(base64 < vex.openvex.json | tr -d '\n')"]
}
EOF
```

Matching statements (e.g. `status: not_affected`) are applied against the
scan results before they're returned, so suppressed vulnerabilities won't
appear in the response.

> [!NOTE]
> Supported SBOM formats include CycloneDX XML and JSON, SPDX and Syft.

## Contributing

If you are ready to jump in and test, add code, or help with documentation,
please follow the instructions on our [contributing guide](/CONTRIBUTING.md)
for details on how to open issues, setup VMClarity for development and test.

## Code of Conduct

You can view our code of conduct [here](/CODE_OF_CONDUCT.md).

## License

[Apache License, Version 2.0](/LICENSE)
