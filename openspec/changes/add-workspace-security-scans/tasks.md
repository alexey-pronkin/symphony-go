## 1. Backend

- [x] 1.1 Add optional workspace scan config and Trivy scanner service
- [x] 1.2 Enrich issue detail responses with cached workspace scan results
- [x] 1.3 Add Go unit coverage for scan parsing and server enrichment

## 2. Frontend

- [x] 2.1 Extend issue detail types with workspace scan data
- [x] 2.2 Render workspace scan summary and top findings in the selected issue panel

## 3. Verification

- [x] 3.1 Run `go test ./...`
- [x] 3.2 Run `npm --prefix libretto test -- --runInBand`
- [x] 3.3 Run `npm --prefix libretto run build`
