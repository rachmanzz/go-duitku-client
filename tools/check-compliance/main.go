package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var violations int

const defaultDocsURL = "https://docs.duitku.com/pop/id/"

func main() {
	sdkDir := flag.String("sdk", "", "SDK source directory")
	docsURL := flag.String("docs", defaultDocsURL, "Duitku POP docs URL")
	flag.Parse()

	sdkAbs, err := filepath.Abs(*sdkDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving SDK path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔍 Duitku POP SDK Compliance Checker (Big Pickle)\n")
	fmt.Printf("   Docs : %s\n", *docsURL)
	fmt.Printf("   SDK  : %s\n\n", sdkAbs)

	docSpec := fetchAndParseDocs(*docsURL)
	sdkInfo := parseSDK(sdkAbs)

	compare(docSpec, sdkInfo)

	if violations > 0 {
		fmt.Printf("\n❌ %d compliance violation(s) found.\n", violations)
		os.Exit(1)
	}
	fmt.Println("\n✅ SDK is fully compliant with the official documentation.")
}

// ─── Data structures ────────────────────────────────────────────────

type DocSpec struct {
	CreateInvoice struct {
		Method          string
		SandboxURL      string
		ProductionURL   string
		EndpointPath    string
		Headers         []DocField
		RequestFields   []DocField
		ResponseFields  []DocField
		SignatureDesc   string
	}
	Callback struct {
		Fields        []DocField
		SignatureDesc string
	}
	SubObjects map[string][]DocField
}

type DocField struct {
	Name        string
	Type        string
	Required    bool
	MaxLen      int
	Description string
	JSONName    string
	GoType      string
}

type SDKInfo struct {
	Structs    map[string]StructInfo
	SigContent string
	InvContent string
	ClContent  string
}

type StructInfo struct {
	Name   string
	Fields []FieldInfo
}

type FieldInfo struct {
	Name      string
	Type      string
	JSONName  string
	OmitEmpty bool
	FormName  string
}

// ─── Docs fetcher & HTML parser ─────────────────────────────────────

func fetchAndParseDocs(url string) *DocSpec {
	fmt.Println("📡 Fetching official documentation...")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("  ⚠️  Cannot fetch docs (%v) — using local cache\n", err)
		return getCachedSpec()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("  ⚠️  Docs returned HTTP %d — using local cache\n", resp.StatusCode)
		return getCachedSpec()
	}

	spec, err := parseDocsHTML(resp.Body)
	if err != nil {
		fmt.Printf("  ⚠️  Cannot parse docs HTML (%v) — using local cache\n", err)
		return getCachedSpec()
	}

	fmt.Println("  ✅ Docs fetched and parsed successfully")
	return spec
}

func getCachedSpec() *DocSpec {
	s := &DocSpec{SubObjects: map[string][]DocField{}}
	// ── Endpoint ────────────────────────────────────────────
	s.CreateInvoice.Method = "POST"
	s.CreateInvoice.EndpointPath = "/api/merchant/createInvoice"
	s.CreateInvoice.SandboxURL = "https://api-sandbox.duitku.com"
	s.CreateInvoice.ProductionURL = "https://api-prod.duitku.com"
	s.CreateInvoice.SignatureDesc = "stringToSign = merchantCode + timestamp; signature = HMAC_SHA256(stringToSign, apiKey)"

	// ── Headers ─────────────────────────────────────────────
	s.CreateInvoice.Headers = []DocField{
		{Name: "Content-Type", Required: true, JSONName: "Content-Type"},
		{Name: "x-duitku-timestamp", Required: true, JSONName: "x-duitku-timestamp"},
		{Name: "x-duitku-signature", Required: true, JSONName: "x-duitku-signature"},
		{Name: "x-duitku-merchantcode", Required: true, JSONName: "x-duitku-merchantcode"},
	}

	// ── Request fields ──────────────────────────────────────
	s.CreateInvoice.RequestFields = []DocField{
		{Name: "paymentAmount", Type: "integer", Required: true, GoType: "int64"},
		{Name: "merchantOrderId", Type: "string(50)", Required: true, MaxLen: 50, GoType: "string"},
		{Name: "productDetails", Type: "string(255)", Required: true, MaxLen: 255, GoType: "string"},
		{Name: "email", Type: "string(255)", Required: true, MaxLen: 255, GoType: "string"},
		{Name: "additionalParam", Type: "string(255)", Required: false, MaxLen: 255, GoType: "string"},
		{Name: "merchantUserInfo", Type: "string(255)", Required: false, MaxLen: 255, GoType: "string"},
		{Name: "customerVaName", Type: "string(20)", Required: false, MaxLen: 20, GoType: "string"},
		{Name: "phoneNumber", Type: "string(50)", Required: false, MaxLen: 50, GoType: "string"},
		{Name: "itemDetails", Type: "itemDetails", Required: false},
		{Name: "customerDetail", Type: "customerDetail", Required: false},
		{Name: "creditCardDetail", Type: "creditCardDetail", Required: false},
		{Name: "callbackUrl", Type: "string(255)", Required: true, MaxLen: 255, GoType: "string"},
		{Name: "returnUrl", Type: "string(255)", Required: true, MaxLen: 255, GoType: "string"},
		{Name: "expiryPeriod", Type: "integer", Required: false, GoType: "int"},
		{Name: "paymentMethod", Type: "string", Required: false, GoType: "string"},
	}

	// ── Response fields ─────────────────────────────────────
	s.CreateInvoice.ResponseFields = []DocField{
		{Name: "merchantCode", GoType: "string"},
		{Name: "reference", GoType: "string"},
		{Name: "paymentUrl", GoType: "string"},
		{Name: "statusCode", GoType: "string"},
		{Name: "statusMessage", GoType: "string"},
	}

	// ── Sub-objects ─────────────────────────────────────────
	s.SubObjects["ItemDetail"] = []DocField{
		{Name: "name", Type: "string(50)", Required: true, MaxLen: 50, GoType: "string"},
		{Name: "price", Type: "integer", Required: true, GoType: "int64"},
		{Name: "quantity", Type: "integer", Required: true, GoType: "int"},
	}
	s.SubObjects["CustomerDetail"] = []DocField{
		{Name: "firstName", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "lastName", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "email", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "phoneNumber", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "billingAddress", Type: "Address"},
		{Name: "shippingAddress", Type: "Address"},
		{Name: "merchantCustomerId", Type: "string(100)", MaxLen: 100, GoType: "string"},
	}
	s.SubObjects["Address"] = []DocField{
		{Name: "firstName", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "lastName", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "address", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "city", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "postalCode", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "phone", Type: "string(50)", MaxLen: 50, GoType: "string"},
		{Name: "countryCode", Type: "string(50)", MaxLen: 50, GoType: "string"},
	}
	s.SubObjects["CreditCardDetail"] = []DocField{
		{Name: "saveCardToken", Type: "int(1)", GoType: "int"},
		{Name: "acquirer", Type: "string(3)", MaxLen: 3, GoType: "string"},
		{Name: "binWhitelist", Type: "array string(6)", GoType: "[]string"},
	}

	// ── Callback ────────────────────────────────────────────
	s.Callback.SignatureDesc = "stringToSign = merchantcode + amount + merchantOrderId; signature = HMAC_SHA256(stringToSign, merchantKey)"
	s.Callback.Fields = []DocField{
		{Name: "merchantCode", GoType: "string"},
		{Name: "amount", GoType: "int64"},
		{Name: "merchantOrderId", GoType: "string"},
		{Name: "productDetail", GoType: "string"},
		{Name: "additionalParam", GoType: "string"},
		{Name: "paymentCode", GoType: "string"},
		{Name: "resultCode", GoType: "string"},
		{Name: "merchantUserId", GoType: "string"},
		{Name: "reference", GoType: "string"},
		{Name: "signature", GoType: "string"},
		{Name: "publisherOrderId", GoType: "string"},
		{Name: "spUserHash", GoType: "string"},
		{Name: "settlementDate", GoType: "string"},
		{Name: "issuerCode", GoType: "string"},
		{Name: "bankAppCode", GoType: "string"},
		{Name: "bankOrderId", GoType: "string"},
		{Name: "bankRespCode", GoType: "string"},
		{Name: "bankRespMsg", GoType: "string"},
		{Name: "cardName", GoType: "string"},
		{Name: "cardType", GoType: "string"},
		{Name: "maskedNumber", GoType: "string"},
		{Name: "tokenId", GoType: "string"},
		{Name: "transactionState", GoType: "string"},
		{Name: "transactionStateStatus", GoType: "string"},
		{Name: "merchantCustomerId", GoType: "string"},
		{Name: "expiryDate", GoType: "string"},
	}
	return s
}

func parseDocsHTML(r io.Reader) (*DocSpec, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("html parse: %w", err)
	}

	s := &DocSpec{SubObjects: map[string][]DocField{}}
	collectText := func(n *html.Node) string {
		var b strings.Builder
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if n.Type == html.TextNode {
				b.WriteString(n.Data)
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(n)
		return strings.TrimSpace(b.String())
	}

	section := ""
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h2", "h3":
				rawText := collectText(n)
				text := strings.ToLower(rawText)
				switch {
				case strings.Contains(text, "create invoice"):
					section = "create-invoice"
				case strings.Contains(text, "obyek json"):
					section = "objects"
				case strings.Contains(text, "callback") && !strings.Contains(text, "js"):
					section = "callback"
				case strings.Contains(text, "item detail"):
					section = "item-detail"
				case strings.Contains(text, "customer detail"):
					section = "customer-detail"
				case strings.Contains(text, "address"):
					if section == "objects" || section == "create-invoice" {
						section = "address"
					}
				case strings.Contains(text, "credit card detail"):
					section = "cc-detail"
				case strings.Contains(text, "respon parameter"):
					section = "response"
				case strings.Contains(text, "request header"):
					section = "headers"
				case strings.Contains(text, "endpoint"):
					if section == "create-invoice" {
						section = "endpoint"
					}
				}
			case "table":
				rows := parseTable(n, collectText)
				if len(rows) < 2 {
					break
				}
				headers := rows[0]
				dataRows := rows[1:]

				switch section {
				case "headers":
					s.CreateInvoice.Headers = parseFieldTable(headers, dataRows)
				case "create-invoice":
					s.CreateInvoice.RequestFields = parseFieldTable(headers, dataRows)
				case "response":
					s.CreateInvoice.ResponseFields = parseFieldTableSimple(headers, dataRows)
				case "callback":
					s.Callback.Fields = parseCallbackFields(headers, dataRows)
				case "item-detail":
					s.SubObjects["ItemDetail"] = parseFieldTableSimple(headers, dataRows)
				case "customer-detail":
					s.SubObjects["CustomerDetail"] = parseFieldTableSimple(headers, dataRows)
				case "address":
					s.SubObjects["Address"] = parseFieldTableSimple(headers, dataRows)
				case "cc-detail":
					s.SubObjects["CreditCardDetail"] = parseFieldTableSimple(headers, dataRows)
				}
			case "p":
				text := collectText(n)
				low := strings.ToLower(text)
				if section == "create-invoice" && strings.Contains(low, "signature") {
					s.CreateInvoice.SignatureDesc = extractSignature(text)
				}
				if section == "callback" && strings.Contains(low, "signature") {
					s.Callback.SignatureDesc = extractSignature(text)
				}
			case "pre":
				code := collectText(n)
				low := strings.ToLower(code)
				if strings.Contains(low, "api-sandbox") && strings.Contains(low, "createinvoice") {
					re := regexp.MustCompile(`https://[^\s'"]+/api/merchant/createInvoice`)
					m := re.FindString(code)
					if strings.Contains(m, "sandbox") {
						s.CreateInvoice.SandboxURL = "https://api-sandbox.duitku.com"
					} else if strings.Contains(m, "prod") {
						s.CreateInvoice.ProductionURL = "https://api-prod.duitku.com"
					}
				}
				if strings.Contains(low, "methodpost") || strings.Contains(low, "method: post") || strings.Contains(low, `"post`) || strings.Contains(low, "method: `post") {
					s.CreateInvoice.Method = "POST"
				}
				if strings.Contains(low, "createinvoice") {
					s.CreateInvoice.EndpointPath = "/api/merchant/createInvoice"
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	fillEndpointFallback(s)

	// If HTML parsing missed some sections, merge with cached spec
	cached := getCachedSpec()
	if len(s.CreateInvoice.Headers) == 0 {
		s.CreateInvoice.Headers = cached.CreateInvoice.Headers
	}
	if len(s.CreateInvoice.RequestFields) == 0 {
		s.CreateInvoice.RequestFields = cached.CreateInvoice.RequestFields
	}
	if len(s.CreateInvoice.ResponseFields) == 0 {
		s.CreateInvoice.ResponseFields = cached.CreateInvoice.ResponseFields
	}
	if len(s.Callback.Fields) == 0 {
		s.Callback.Fields = cached.Callback.Fields
	}
	for _, k := range []string{"ItemDetail", "CustomerDetail", "Address", "CreditCardDetail"} {
		if len(s.SubObjects[k]) == 0 {
			if v, ok := cached.SubObjects[k]; ok {
				s.SubObjects[k] = v
			}
		}
	}

	return s, nil
}

func fillEndpointFallback(s *DocSpec) {
	if s.CreateInvoice.Method == "" {
		s.CreateInvoice.Method = "POST"
	}
	if s.CreateInvoice.EndpointPath == "" {
		s.CreateInvoice.EndpointPath = "/api/merchant/createInvoice"
	}
	if s.CreateInvoice.SandboxURL == "" {
		s.CreateInvoice.SandboxURL = "https://api-sandbox.duitku.com"
	}
	if s.CreateInvoice.ProductionURL == "" {
		s.CreateInvoice.ProductionURL = "https://api-prod.duitku.com"
	}
}

func parseTable(n *html.Node, textFn func(*html.Node) string) [][]string {
	var rows [][]string
	var curRow []string
	skip := 0

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if skip > 0 {
			skip--
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "tr":
				if len(curRow) > 0 {
					rows = append(rows, curRow)
				}
				curRow = nil
			case "th", "td":
				curRow = append(curRow, textFn(n))
			case "table":
				// nested table — skip
				skip = countNodes(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	if len(curRow) > 0 {
		rows = append(rows, curRow)
	}
	return rows
}

func countNodes(n *html.Node) int {
	c := 1
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		c += countNodes(ch)
	}
	return c
}

func findColIndex(headers []string, names ...string) int {
	for _, name := range names {
		for i, h := range headers {
			if strings.EqualFold(strings.TrimSpace(h), name) {
				return i
			}
		}
	}
	return -1
}

func parseFieldTable(headers []string, rows [][]string) []DocField {
	nameIdx := findColIndex(headers, "parameter", "nama")
	typeIdx := findColIndex(headers, "tipe")
	reqIdx := findColIndex(headers, "wajib")
	descIdx := findColIndex(headers, "deskripsi", "keterangan")

	var fields []DocField
	for _, row := range rows {
		f := DocField{}
		if nameIdx >= 0 && nameIdx < len(row) {
			f.Name = strings.TrimSpace(row[nameIdx])
		}
		if typeIdx >= 0 && typeIdx < len(row) {
			f.Type = strings.TrimSpace(row[typeIdx])
			f.GoType = docTypeToGoType(f.Type)
			f.MaxLen = extractMaxLen(f.Type)
		}
		if reqIdx >= 0 && reqIdx < len(row) {
			f.Required = strings.Contains(strings.ToLower(row[reqIdx]), "✓")
		}
		if descIdx >= 0 && descIdx < len(row) {
			f.Description = strings.TrimSpace(row[descIdx])
		}
		if f.Name != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func parseFieldTableSimple(headers []string, rows [][]string) []DocField {
	nameIdx := findColIndex(headers, "parameter", "nama")
	typeIdx := findColIndex(headers, "tipe")
	reqIdx := findColIndex(headers, "wajib")

	var fields []DocField
	for _, row := range rows {
		f := DocField{}
		if nameIdx >= 0 && nameIdx < len(row) {
			f.Name = strings.TrimSpace(row[nameIdx])
		}
		if typeIdx >= 0 && typeIdx < len(row) {
			f.Type = strings.TrimSpace(row[typeIdx])
			f.GoType = docTypeToGoType(f.Type)
			f.MaxLen = extractMaxLen(f.Type)
		}
		if reqIdx >= 0 && reqIdx < len(row) {
			f.Required = strings.Contains(strings.ToLower(row[reqIdx]), "✓")
		}
		if f.Name != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func parseCallbackFields(headers []string, rows [][]string) []DocField {
	nameIdx := findColIndex(headers, "parameter", "nama")
	descIdx := findColIndex(headers, "deskripsi")

	var fields []DocField
	for _, row := range rows {
		f := DocField{}
		if nameIdx >= 0 && nameIdx < len(row) {
			f.Name = strings.TrimSpace(row[nameIdx])
		}
		if descIdx >= 0 && descIdx < len(row) {
			f.Description = strings.TrimSpace(row[descIdx])
		}
		f.GoType = "string"
		if f.Name == "amount" {
			f.GoType = "int64"
		}
		if f.Name != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func docTypeToGoType(dt string) string {
	dt = strings.ToLower(strings.TrimSpace(dt))
	switch {
	case dt == "integer", dt == "int", strings.HasPrefix(dt, "int("):
		return "int64"
	case dt == "string", strings.HasPrefix(dt, "string("):
		return "string"
	case strings.HasPrefix(dt, "array"):
		return "[]string"
	case strings.Contains(dt, "itemdetail"):
		return "[]ItemDetail"
	case strings.Contains(dt, "customerdetail"):
		return "*CustomerDetail"
	case strings.Contains(dt, "creditcarddetail"):
		return "*CreditCardDetail"
	case strings.Contains(dt, "address"):
		return "*Address"
	}
	return "string"
}

func extractMaxLen(dt string) int {
	re := regexp.MustCompile(`\((\d+)\)`)
	m := re.FindStringSubmatch(dt)
	if len(m) > 1 {
		var n int
		fmt.Sscanf(m[1], "%d", &n)
		return n
	}
	return 0
}

func extractSignature(text string) string {
	re := regexp.MustCompile(`(?s)stringToSign\s*=\s*[^.]+\.`)
	m := re.FindString(text)
	if m != "" {
		return strings.TrimSpace(m)
	}
	return strings.TrimSpace(text)
}

// ─── SDK parser ─────────────────────────────────────────────────────

func parseSDK(sdkDir string) *SDKInfo {
	info := &SDKInfo{
		Structs: make(map[string]StructInfo),
	}

	filepath.Walk(sdkDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(src)

		if strings.HasSuffix(path, "signature.go") {
			info.SigContent = content
		}
		if strings.HasSuffix(path, "invoice.go") {
			info.InvContent = content
		}
		if strings.HasSuffix(path, "callback.go") {
			info.ClContent = content
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				si := StructInfo{Name: ts.Name.Name}
				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						continue
					}
					fi := FieldInfo{Name: field.Names[0].Name, Type: exprToString(field.Type)}
					if field.Tag != nil {
						tag := strings.Trim(field.Tag.Value, "`")
						fi.JSONName, fi.OmitEmpty = parseTag(tag, "json")
						fi.FormName, _ = parseTag(tag, "form")
					}
					si.Fields = append(si.Fields, fi)
				}
				info.Structs[si.Name] = si
			}
		}
		return nil
	})

	return info
}

func exprToString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return fmt.Sprintf("[%s]%s", exprToString(t.Len), exprToString(t.Elt))
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", t)
	}
}

func parseTag(tag, key string) (string, bool) {
	for _, part := range strings.Fields(tag) {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 && kv[0] == key {
			val := strings.Trim(kv[1], "\"")
			parts := strings.Split(val, ",")
			return parts[0], len(parts) > 1
		}
	}
	return "", false
}

// ─── Comparator ─────────────────────────────────────────────────────

func fail(format string, args ...interface{}) {
	violations++
	fmt.Printf("  ❌ "+format+"\n", args...)
}

func pass(format string, args ...interface{}) {
	fmt.Printf("  ✅ "+format+"\n", args...)
}

func warn(format string, args ...interface{}) {
	fmt.Printf("  ⚠️  "+format+"\n", args...)
}

func compare(doc *DocSpec, sdk *SDKInfo) {
	// ── 1. CreateInvoice endpoint & method ──────────────────
	fmt.Println("🌐 API Endpoint:")
	checkEndpoint(doc, sdk)

	// ── 2. Headers ─────────────────────────────────────────
	fmt.Println("\n📨 Request Headers:")
	checkHeaders(doc, sdk)

	// ── 3. Request fields ──────────────────────────────────
	fmt.Println("\n📋 CreateInvoiceRequest fields:")
	checkRequestFields(doc, sdk)

	// ── 4. Response fields ─────────────────────────────────
	fmt.Println("\n📋 CreateInvoiceResponse fields:")
	checkResponseFields(doc, sdk)

	// ── 5. Sub-objects ─────────────────────────────────────
	for _, name := range []string{"ItemDetail", "CustomerDetail", "Address", "CreditCardDetail"} {
		fmt.Printf("\n📋 %s fields:\n", name)
		checkSubObject(doc, sdk, name)
	}

	// ── 6. Callback fields ─────────────────────────────────
	fmt.Println("\n📋 CallbackRequest fields:")
	checkCallbackFields(doc, sdk)

	// ── 7. Signature algorithms ────────────────────────────
	fmt.Println("\n🔐 Signature algorithms:")
	checkSignatures(doc, sdk)
}

func checkEndpoint(doc *DocSpec, sdk *SDKInfo) {
	if strings.Contains(sdk.InvContent, "MethodPost") {
		pass("Uses HTTP POST (%s)", doc.CreateInvoice.Method)
	} else {
		fail("Should use HTTP POST (%s)", doc.CreateInvoice.Method)
	}
	if strings.Contains(sdk.InvContent, doc.CreateInvoice.EndpointPath) {
		pass("Endpoint path: %s", doc.CreateInvoice.EndpointPath)
	} else {
		fail("Endpoint path should be %s", doc.CreateInvoice.EndpointPath)
	}
	// Check for base URL constants in duitku.go
	sdkDir := findSDKDir()
	if sdkDir != "" {
		duitkuSrc, err := os.ReadFile(filepath.Join(sdkDir, "duitku.go"))
		if err == nil {
			dc := string(duitkuSrc)
			if strings.Contains(dc, "api-sandbox.duitku.com") {
				pass("Sandbox URL constant found")
			} else {
				fail("Sandbox URL constant missing")
			}
			if strings.Contains(dc, "api-prod.duitku.com") {
				pass("Production URL constant found")
			} else {
				fail("Production URL constant missing")
			}
		}
	}
	if strings.Contains(sdk.InvContent, "BaseURL(") {
		pass("Base URL resolved dynamically via BaseURL()")
	}
}

func findSDKDir() string {
	for _, d := range []string{".", "..", "../.."} {
		p := filepath.Join(d, "duitku.go")
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(d)
			return abs
		}
	}
	return ""
}

func checkHeaders(doc *DocSpec, sdk *SDKInfo) {
	for _, h := range doc.CreateInvoice.Headers {
		if strings.Contains(sdk.InvContent, `"`+h.JSONName+`"`) {
			pass("Header %s is set", h.JSONName)
		} else {
			fail("Header %s is not set", h.JSONName)
		}
	}
}

func checkRequestFields(doc *DocSpec, sdk *SDKInfo) {
	si, ok := sdk.Structs["CreateInvoiceRequest"]
	if !ok {
		fail("Struct CreateInvoiceRequest not found")
		return
	}

	for _, df := range doc.CreateInvoice.RequestFields {
		field := findFieldByJSON(si, df.Name)
		if field == nil {
			fail("Missing JSON field %q", df.Name)
			continue
		}

		if df.GoType != "" && field.Type != df.GoType {
			warn("Field %s: Go type %q, docs suggest %q", field.Name, field.Type, df.GoType)
		}

		if df.Required && field.JSONName != "" && field.OmitEmpty {
			fail("Required field %s has omitempty — should always be sent", field.Name)
		}
		if !df.Required && !field.OmitEmpty && field.JSONName != "" {
			warn("Optional field %s lacks omitempty — will send zero-value", field.Name)
		}
	}
	pass("All request fields present")
}

func checkResponseFields(doc *DocSpec, sdk *SDKInfo) {
	si, ok := sdk.Structs["CreateInvoiceResponse"]
	if !ok {
		fail("Struct CreateInvoiceResponse not found")
		return
	}
	for _, df := range doc.CreateInvoice.ResponseFields {
		field := findFieldByJSON(si, df.Name)
		if field == nil {
			fail("Missing response field %q", df.Name)
		}
	}
	pass("All response fields present")
}

func checkSubObject(doc *DocSpec, sdk *SDKInfo, name string) {
	si, ok := sdk.Structs[name]
	if !ok {
		fail("Struct %s not found", name)
		return
	}
	expected, ok := doc.SubObjects[name]
	if !ok {
		warn("No docs spec for %s — skipping", name)
		return
	}
	for _, df := range expected {
		field := findFieldByJSON(si, df.Name)
		if field == nil {
			fail("Missing field %q in %s", df.Name, name)
			continue
		}
		if df.GoType != "" && field.Type != df.GoType {
			warn("Field %s.%s: Go type %q, docs suggest %q", name, field.Name, field.Type, df.GoType)
		}
	}
	pass("All fields present")
}

func checkCallbackFields(doc *DocSpec, sdk *SDKInfo) {
	si, ok := sdk.Structs["CallbackRequest"]
	if !ok {
		fail("Struct CallbackRequest not found")
		return
	}
	for _, df := range doc.Callback.Fields {
		field := findFieldByName(si, toGoName(df.Name))
		if field == nil {
			field = findFieldByForm(si, df.Name)
		}
		if field == nil {
			fail("Missing field %q in CallbackRequest", df.Name)
			continue
		}
		if df.GoType != "" && field.Type != df.GoType {
			warn("Field %s: Go type %q, docs suggest %q", field.Name, field.Type, df.GoType)
		}
		if field.FormName != df.Name {
			warn("Field %s: form tag %q, docs parameter %q", field.Name, field.FormName, df.Name)
		}
	}
	pass("All callback fields present")
}

func checkSignatures(doc *DocSpec, sdk *SDKInfo) {
	// generateSignature
	if strings.Contains(sdk.SigContent, "merchantCode + timestamp") {
		pass("generateSignature: stringToSign = merchantCode + timestamp")
	} else {
		fail("generateSignature should use stringToSign = merchantCode + timestamp")
	}
	if strings.Contains(sdk.SigContent, "hmac.New(sha256.New") {
		pass("generateSignature: HMAC SHA256")
	} else {
		fail("generateSignature should use HMAC SHA256")
	}
	if strings.Contains(sdk.SigContent, "hex.EncodeToString") {
		pass("generateSignature: hex-encoded output")
	} else {
		fail("generateSignature should output hex-encoded")
	}

	// VerifyCallbackSignature
	if strings.Contains(sdk.SigContent, "merchantCode") && strings.Contains(sdk.SigContent, "amount") && strings.Contains(sdk.SigContent, "merchantOrderID") {
		pass("VerifyCallbackSignature: stringToSign = merchantCode + amount + merchantOrderId")
	} else {
		fail("VerifyCallbackSignature should use merchantCode + amount + merchantOrderId")
	}
	if strings.Contains(sdk.SigContent, "hmac.Equal") {
		pass("VerifyCallbackSignature: constant-time comparison")
	} else {
		fail("VerifyCallbackSignature should use constant-time comparison")
	}
}

// ─── Helpers ────────────────────────────────────────────────────────

func findFieldByJSON(si StructInfo, jsonName string) *FieldInfo {
	for i := range si.Fields {
		if si.Fields[i].JSONName == jsonName {
			return &si.Fields[i]
		}
	}
	return nil
}

func findFieldByName(si StructInfo, name string) *FieldInfo {
	for i := range si.Fields {
		if si.Fields[i].Name == name {
			return &si.Fields[i]
		}
	}
	return nil
}

func findFieldByForm(si StructInfo, formName string) *FieldInfo {
	for i := range si.Fields {
		if si.Fields[i].FormName == formName {
			return &si.Fields[i]
		}
	}
	return nil
}

func toGoName(s string) string {
	parts := strings.Split(s, "_")
	var result string
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.EqualFold(p, "id") {
			result += "ID"
		} else {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}
