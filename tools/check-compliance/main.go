package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var violations int

func main() {
	sdkDir := flag.String("sdk", "", "path to SDK source directory (default: parent of tools/check-compliance)")
	flag.Parse()

	if *sdkDir == "" {
		execPath, _ := os.Executable()
		*sdkDir = filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
	}

	sdkAbs, err := filepath.Abs(*sdkDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔍 Duitku POP SDK Compliance Checker (Big Pickle)\n")
	fmt.Printf("   SDK directory: %s\n\n", sdkAbs)

	structs := parseStructs(sdkAbs)

	checkCreateInvoiceRequest(structs)
	checkCreateInvoiceResponse(structs)
	checkCustomerDetail(structs)
	checkAddress(structs)
	checkItemDetail(structs)
	checkCreditCardDetail(structs)
	checkCallbackRequest(structs)
	checkSignatureAlgorithms(sdkAbs)
	checkEndpoints(sdkAbs)
	checkHeaders(sdkAbs)

	if violations > 0 {
		fmt.Printf("\n❌ %d compliance violation(s) found.\n", violations)
		os.Exit(1)
	}
	fmt.Println("\n✅ SDK is fully compliant with the official documentation.")
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
	HasForm   bool
	FormName  string
}

func parseStructs(sdkDir string) map[string]StructInfo {
	result := make(map[string]StructInfo)

	err := filepath.Walk(sdkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: error parsing %s: %v\n", path, err)
			return nil
		}

		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				si := StructInfo{Name: typeSpec.Name.Name}
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						continue
					}
					fi := FieldInfo{
						Name: field.Names[0].Name,
						Type: exprToString(field.Type),
					}
					if field.Tag != nil {
						tag := field.Tag.Value
						tag = strings.Trim(tag, "`")
						fi.JSONName, fi.OmitEmpty = parseJSONTag(tag)
						fi.FormName, fi.HasForm = parseFormTag(tag)
					}
					si.Fields = append(si.Fields, fi)
				}
				result[si.Name] = si
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error walking SDK directory: %v\n", err)
		os.Exit(1)
	}

	return result
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return "[" + exprToString(t.Len) + "]" + exprToString(t.Elt)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.BasicLit:
		return t.Value
	default:
		return fmt.Sprintf("%T", t)
	}
}

func parseJSONTag(tag string) (name string, omitempty bool) {
	return parseTagOption(tag, "json")
}

func parseFormTag(tag string) (name string, has bool) {
	return parseTagOption(tag, "form")
}

func parseTagOption(tag, key string) (string, bool) {
	for _, part := range strings.Fields(tag) {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 && kv[0] == key {
			val := strings.Trim(kv[1], "\"")
			parts := strings.Split(val, ",")
			name := parts[0]
			hasOpt := len(parts) > 1
			return name, hasOpt
		}
	}
	return "", false
}

func fail(format string, args ...interface{}) {
	violations++
	fmt.Printf("  ❌ "+format+"\n", args...)
}

func pass(format string, args ...interface{}) {
	fmt.Printf("  ✅ "+format+"\n", args...)
}

func findField(si StructInfo, jsonName string) *FieldInfo {
	for i := range si.Fields {
		if si.Fields[i].JSONName == jsonName {
			return &si.Fields[i]
		}
	}
	return nil
}

func checkStructField(si StructInfo, name, expectedType, jsonName string, required bool) {
	field := findField(si, jsonName)
	if field == nil {
		fail("Struct %s: missing field with JSON name %q", si.Name, jsonName)
		return
	}
	if field.Name != name {
		fail("Struct %s: field %q has Go name %q, expected %q", si.Name, jsonName, field.Name, name)
	}
	if field.Type != expectedType && !typeEquivalent(field.Type, expectedType) {
		fail("Struct %s: field %s has type %q, expected %q", si.Name, field.Name, field.Type, expectedType)
	}
	if required && field.OmitEmpty {
		fail("Struct %s: required field %s has omitempty, should not", si.Name, field.Name)
	}
	if !required && !field.OmitEmpty {
		fail("Struct %s: optional field %s should have omitempty", si.Name, field.Name)
	}
}

func typeEquivalent(actual, expected string) bool {
	if actual == expected {
		return true
	}
	return false
}

// -------------------- Check Functions --------------------

func checkCreateInvoiceRequest(structs map[string]StructInfo) {
	fmt.Println("📋 CreateInvoiceRequest fields:")

	si, ok := structs["CreateInvoiceRequest"]
	if !ok {
		fail("Struct CreateInvoiceRequest not found")
		return
	}

	type fieldDef struct {
		name       string
		typ        string
		jsonName   string
		required   bool
	}

	expected := []fieldDef{
		{"PaymentAmount", "int64", "paymentAmount", true},
		{"MerchantOrderID", "string", "merchantOrderId", true},
		{"ProductDetails", "string", "productDetails", true},
		{"Email", "string", "email", true},
		{"PhoneNumber", "string", "phoneNumber", false},
		{"AdditionalParam", "string", "additionalParam", false},
		{"MerchantUserInfo", "string", "merchantUserInfo", false},
		{"CustomerVaName", "string", "customerVaName", false},
		{"PaymentMethod", "string", "paymentMethod", false},
		{"ItemDetails", "[]ItemDetail", "itemDetails", false},
		{"CustomerDetail", "*CustomerDetail", "customerDetail", false},
		{"CreditCardDetail", "*CreditCardDetail", "creditCardDetail", false},
		{"CallbackURL", "string", "callbackUrl", true},
		{"ReturnURL", "string", "returnUrl", true},
		{"ExpiryPeriod", "int", "expiryPeriod", false},
	}

	for _, f := range expected {
		checkStructField(si, f.name, f.typ, f.jsonName, f.required)
	}

	pass("All fields present and correctly typed")
}

func checkCreateInvoiceResponse(structs map[string]StructInfo) {
	fmt.Println("\n📋 CreateInvoiceResponse fields:")

	si, ok := structs["CreateInvoiceResponse"]
	if !ok {
		fail("Struct CreateInvoiceResponse not found")
		return
	}

	type fieldDef struct {
		name     string
		typ      string
		jsonName string
	}

	expected := []fieldDef{
		{"MerchantCode", "string", "merchantCode"},
		{"Reference", "string", "reference"},
		{"PaymentURL", "string", "paymentUrl"},
		{"StatusCode", "string", "statusCode"},
		{"StatusMessage", "string", "statusMessage"},
	}

	for _, f := range expected {
		field := findField(si, f.jsonName)
		if field == nil {
			fail("Missing field with JSON name %q", f.jsonName)
			continue
		}
		if field.Name != f.name {
			fail("Field %q has Go name %q, expected %q", f.jsonName, field.Name, f.name)
		}
		if field.Type != f.typ {
			fail("Field %s has type %q, expected %q", field.Name, field.Type, f.typ)
		}
	}
	pass("All fields present and correctly typed")
}

func checkCustomerDetail(structs map[string]StructInfo) {
	fmt.Println("\n📋 CustomerDetail fields:")

	si, ok := structs["CustomerDetail"]
	if !ok {
		fail("Struct CustomerDetail not found")
		return
	}

	type fieldDef struct {
		name     string
		typ      string
		jsonName string
		required bool
	}

	expected := []fieldDef{
		{"FirstName", "string", "firstName", false},
		{"LastName", "string", "lastName", false},
		{"Email", "string", "email", false},
		{"PhoneNumber", "string", "phoneNumber", false},
		{"BillingAddress", "*Address", "billingAddress", false},
		{"ShippingAddress", "*Address", "shippingAddress", false},
		{"MerchantCustomerID", "string", "merchantCustomerId", false},
	}

	for _, f := range expected {
		checkStructField(si, f.name, f.typ, f.jsonName, f.required)
	}
	pass("All fields present and correctly typed")
}

func checkAddress(structs map[string]StructInfo) {
	fmt.Println("\n📋 Address fields:")

	si, ok := structs["Address"]
	if !ok {
		fail("Struct Address not found")
		return
	}

	type fieldDef struct {
		name     string
		typ      string
		jsonName string
	}

	expected := []fieldDef{
		{"FirstName", "string", "firstName"},
		{"LastName", "string", "lastName"},
		{"Address", "string", "address"},
		{"City", "string", "city"},
		{"PostalCode", "string", "postalCode"},
		{"Phone", "string", "phone"},
		{"CountryCode", "string", "countryCode"},
	}

	for _, f := range expected {
		field := findField(si, f.jsonName)
		if field == nil {
			fail("Missing field with JSON name %q", f.jsonName)
			continue
		}
		if field.Name != f.name {
			fail("Field %q has Go name %q, expected %q", f.jsonName, field.Name, f.name)
		}
		if field.Type != f.typ {
			fail("Field %s has type %q, expected %q", field.Name, field.Type, f.typ)
		}
	}
	pass("All fields present and correctly typed")
}

func checkItemDetail(structs map[string]StructInfo) {
	fmt.Println("\n📋 ItemDetail fields:")

	si, ok := structs["ItemDetail"]
	if !ok {
		fail("Struct ItemDetail not found")
		return
	}

	type fieldDef struct {
		name     string
		typ      string
		jsonName string
	}

	expected := []fieldDef{
		{"Name", "string", "name"},
		{"Price", "int64", "price"},
		{"Quantity", "int", "quantity"},
	}

	for _, f := range expected {
		field := findField(si, f.jsonName)
		if field == nil {
			fail("Missing field with JSON name %q", f.jsonName)
			continue
		}
		if field.Name != f.name {
			fail("Field %q has Go name %q, expected %q", f.jsonName, field.Name, f.name)
		}
		if field.Type != f.typ {
			fail("Field %s has type %q, expected %q", field.Name, field.Type, f.typ)
		}
	}
	pass("All fields present and correctly typed")
}

func checkCreditCardDetail(structs map[string]StructInfo) {
	fmt.Println("\n📋 CreditCardDetail fields:")

	si, ok := structs["CreditCardDetail"]
	if !ok {
		fail("Struct CreditCardDetail not found")
		return
	}

	type fieldDef struct {
		name     string
		typ      string
		jsonName string
	}

	expected := []fieldDef{
		{"SaveCardToken", "int", "saveCardToken"},
		{"Acquirer", "string", "acquirer"},
		{"BinWhitelist", "[]string", "binWhitelist"},
	}

	for _, f := range expected {
		field := findField(si, f.jsonName)
		if field == nil {
			fail("Missing field with JSON name %q", f.jsonName)
			continue
		}
		if field.Name != f.name {
			fail("Field %q has Go name %q, expected %q", f.jsonName, field.Name, f.name)
		}
		if field.Type != f.typ {
			fail("Field %s has type %q, expected %q", field.Name, field.Type, f.typ)
		}
	}
	pass("All fields present and correctly typed")
}

func checkCallbackRequest(structs map[string]StructInfo) {
	fmt.Println("\n📋 CallbackRequest fields:")

	si, ok := structs["CallbackRequest"]
	if !ok {
		fail("Struct CallbackRequest not found")
		return
	}

	type fieldDef struct {
		name     string
		typ      string
		formName string
	}

	expected := []fieldDef{
		{"MerchantCode", "string", "merchantCode"},
		{"Amount", "int64", "amount"},
		{"MerchantOrderID", "string", "merchantOrderId"},
		{"ProductDetail", "string", "productDetail"},
		{"AdditionalParam", "string", "additionalParam"},
		{"PaymentCode", "string", "paymentCode"},
		{"ResultCode", "string", "resultCode"},
		{"MerchantUserID", "string", "merchantUserId"},
		{"Reference", "string", "reference"},
		{"Signature", "string", "signature"},
		{"PublisherOrderID", "string", "publisherOrderId"},
		{"SpUserHash", "string", "spUserHash"},
		{"SettlementDate", "string", "settlementDate"},
		{"IssuerCode", "string", "issuerCode"},
		{"BankAppCode", "string", "bankAppCode"},
		{"BankOrderID", "string", "bankOrderId"},
		{"BankRespCode", "string", "bankRespCode"},
		{"BankRespMsg", "string", "bankRespMsg"},
		{"CardName", "string", "cardName"},
		{"CardType", "string", "cardType"},
		{"MaskedNumber", "string", "maskedNumber"},
		{"TokenID", "string", "tokenId"},
		{"TransactionState", "string", "transactionState"},
		{"TransactionStateStatus", "string", "transactionStateStatus"},
		{"MerchantCustomerID", "string", "merchantCustomerId"},
		{"ExpiryDate", "string", "expiryDate"},
	}

	for _, f := range expected {
		field := findCallbackField(si, f.name)
		if field == nil {
			fail("Struct CallbackRequest: missing field %s", f.name)
			continue
		}
		if field.Type != f.typ {
			fail("Struct CallbackRequest: field %s has type %q, expected %q", field.Name, field.Type, f.typ)
		}
		if field.FormName != f.formName {
			fail("Struct CallbackRequest: field %s has form tag %q, expected %q", field.Name, field.FormName, f.formName)
		}
	}

	pass("All 26 CallbackRequest fields present and correctly typed")
}

func findCallbackField(si StructInfo, name string) *FieldInfo {
	for i := range si.Fields {
		if si.Fields[i].Name == name {
			return &si.Fields[i]
		}
	}
	return nil
}

func checkSignatureAlgorithms(sdkDir string) {
	fmt.Println("\n🔐 Signature algorithms:")

	sigFile := filepath.Join(sdkDir, "signature.go")
	src, err := os.ReadFile(sigFile)
	if err != nil {
		fail("Could not read signature.go: %v", err)
		return
	}
	content := string(src)

	if strings.Contains(content, "merchantCode + timestamp") && strings.Contains(content, "HMAC_SHA256") {
		pass("generateSignature uses stringToSign = merchantCode + timestamp, HMAC SHA256")
	} else if strings.Contains(content, "merchantCode + timestamp") {
		pass("generateSignature uses stringToSign = merchantCode + timestamp")
		if strings.Contains(content, "hmac.New(sha256.New") {
			pass("generateSignature uses HMAC SHA256")
		} else {
			fail("generateSignature should use HMAC SHA256")
		}
	} else {
		fail("generateSignature should use stringToSign = merchantCode + timestamp")
	}

	if strings.Contains(content, "hex.EncodeToString") {
		pass("generateSignature outputs hex-encoded signature")
	} else {
		fail("generateSignature should output hex-encoded lowercase signature")
	}

	callbackCheck := false
	if strings.Contains(content, `Sprintf("%s%d%s"`) || strings.Contains(content, `merchantCode`) {
		if strings.Contains(content, "merchantCode") && strings.Contains(content, "amount") && strings.Contains(content, "merchantOrderID") {
			pass("VerifyCallbackSignature uses stringToSign = merchantCode + amount + merchantOrderId")
			callbackCheck = true
		}
	}
	if !callbackCheck {
		fail("VerifyCallbackSignature should use stringToSign = merchantCode + amount + merchantOrderId")
	}

	if strings.Contains(content, "hmac.Equal") {
		pass("VerifyCallbackSignature uses constant-time comparison (hmac.Equal)")
	} else {
		fail("VerifyCallbackSignature should use constant-time comparison (hmac.Equal)")
	}
}

func checkEndpoints(sdkDir string) {
	fmt.Println("\n🌐 API Endpoints:")

	duitkuSrc, err := os.ReadFile(filepath.Join(sdkDir, "duitku.go"))
	if err != nil {
		fail("Could not read duitku.go: %v", err)
		return
	}
	duitkuContent := string(duitkuSrc)

	sandboxOk := strings.Contains(duitkuContent, "api-sandbox.duitku.com")
	prodOk := strings.Contains(duitkuContent, "api-prod.duitku.com")

	if sandboxOk {
		pass("Sandbox URL constant exists (https://api-sandbox.duitku.com)")
	} else {
		fail("Sandbox URL constant missing or incorrect")
	}
	if prodOk {
		pass("Production URL constant exists (https://api-prod.duitku.com)")
	} else {
		fail("Production URL constant missing or incorrect")
	}

	invoiceSrc, err := os.ReadFile(filepath.Join(sdkDir, "invoice.go"))
	if err != nil {
		fail("Could not read invoice.go: %v", err)
		return
	}
	invoiceContent := string(invoiceSrc)

	if strings.Contains(invoiceContent, "createInvoice") {
		pass("CreateInvoice endpoint is /api/merchant/createInvoice")
	} else {
		fail("CreateInvoice endpoint should be /api/merchant/createInvoice")
	}
	if strings.Contains(invoiceContent, "MethodPost") {
		pass("CreateInvoice uses HTTP POST method")
	} else {
		fail("CreateInvoice should use HTTP POST method")
	}
}

func checkHeaders(sdkDir string) {
	fmt.Println("\n📨 Request Headers:")

	invoiceFile := filepath.Join(sdkDir, "invoice.go")
	src, err := os.ReadFile(invoiceFile)
	if err != nil {
		fail("Could not read invoice.go: %v", err)
		return
	}
	content := string(src)

	checks := []struct {
		header    string
		present   bool
		label     string
	}{
		{"Content-Type", strings.Contains(content, `"Content-Type"`), "Content-Type: application/json"},
		{"x-duitku-signature", strings.Contains(content, `"x-duitku-signature"`), "x-duitku-signature"},
		{"x-duitku-timestamp", strings.Contains(content, `"x-duitku-timestamp"`), "x-duitku-timestamp"},
		{"x-duitku-merchantcode", strings.Contains(content, `"x-duitku-merchantcode"`), "x-duitku-merchantcode"},
	}

	for _, c := range checks {
		if c.present {
			pass("%s header is set", c.label)
		} else {
			fail("%s header is not set", c.label)
		}
	}
}
