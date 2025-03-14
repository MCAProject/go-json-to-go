package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	goformat "go/format"
	"log"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
)

/*
Converted from javascript [json-to-go](https://github.com/mholt/json-to-go) to golang.
*/

var tabs int
var seen map[string][]string
var stack []string
var innerTabs int
var parent string
var globallySeenTypeNames []string
var previousParents string
var accumulator string
var goResult string

// JsonToGoOutput represents the output of the jsonToGo function.
type JsonToGoOutput struct {
	Go    string
	Error string
}

// Helper: checks if a byte is a digit.
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// Helper: checks if a slice of strings contains a string.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// indent adds tabs to goResult.
func indent(tabs int) {
	for i := 0; i < tabs; i++ {
		goResult += "\t"
	}
}

// append appends the string to the global goResult.
func appendFunc(str string) {
	goResult += str
}

// indenter appends tabs to the last element of stack.
func indenter(tabs int) {
	for i := 0; i < tabs; i++ {
		stack[len(stack)-1] += "\t"
	}
}

// appender appends the string to the last element of stack.
func appender(str string) {
	stack[len(stack)-1] += str
}

// toProperCase converts the string to proper case.
func toProperCase(str string) string {
	if match, _ := regexp.MatchString("^[_A-Z0-9]+$", str); match {
		str = strings.ToLower(str)
	}
	commonInitialisms := []string{"ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "UID", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS"}
	reFunc := func(match []string) string {
		sep := match[1]
		frag := match[2]
		if containsStr(commonInitialisms, strings.ToUpper(frag)) {
			return sep + strings.ToUpper(frag)
		}
		if len(frag) > 0 {
			return sep + strings.ToUpper(frag[:1]) + strings.ToLower(frag[1:])
		}
		return sep
	}
	// Replace regex pattern.
	re := regexp.MustCompile(`(^|[^a-zA-Z])([a-z]+)`)
	str = re.ReplaceAllStringFunc(str, func(s string) string {
		parts := re.FindStringSubmatch(s)
		return reFunc(parts)
	})
	re2 := regexp.MustCompile(`([A-Z])([a-z]+)`)
	str = re2.ReplaceAllStringFunc(str, func(s string) string {
		parts := re2.FindStringSubmatch(s)
		if containsStr(commonInitialisms, parts[1]+strings.ToUpper(parts[2])) {
			return strings.ToUpper(parts[1] + parts[2])
		}
		return parts[1] + parts[2]
	})
	return str
}

// uniqueTypeName returns a unique type name based on the given name and seen list.
func uniqueTypeName(name string, seenList []string, prefix ...string) string {
	unique := name
	if !contains(seenList, unique) {
		return unique
	}
	if len(prefix) > 0 && prefix[0] != "" {
		unique = prefix[0] + name
		if !contains(seenList, unique) {
			return unique
		}
	}
	i := 0
	for {
		newName := unique + strconv.Itoa(i)
		if !contains(seenList, newName) {
			return newName
		}
		i++
	}
}

// formatNumber processes numbers in strings.
func formatNumber(str string) string {
	if str == "" {
		return ""
	} else if matched, _ := regexp.MatchString("^\\d+$", str); matched {
		str = "Num" + str
	} else if len(str) > 0 && isDigit(str[0]) {
		numbers := map[byte]string{
			'0': "Zero_",
			'1': "One_",
			'2': "Two_",
			'3': "Three_",
			'4': "Four_",
			'5': "Five_",
			'6': "Six_",
			'7': "Seven_",
			'8': "Eight_",
			'9': "Nine_",
		}
		if val, ok := numbers[str[0]]; ok {
			str = val + str[1:]
		}
	}
	return str
}

// format applies formatting rules to the string.
func format(str string) string {
	str = formatNumber(str)
	// toProperCase: if string is all underscores, capitals or digits, to lower then proper-case.
	sanitized := toProperCase(str)
	// Remove any non-alphanumeric characters.
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")
	sanitized = reg.ReplaceAllString(sanitized, "")
	if sanitized == "" {
		return "NAMING_FAILED"
	}
	return formatNumber(sanitized)
}

// goType returns the Go type corresponding to the given value.
func goType(val interface{}) string {
	if val == nil {
		return "any"
	}
	switch v := val.(type) {
	case string:
		// Check for ISO date format.
		match, _ := regexp.MatchString("^\\d{4}-\\d\\d-\\d\\dT\\d\\d:\\d\\d:\\d\\d(\\.\\d+)?(\\+\\d\\d:\\d\\d|Z)$", v)
		if match {
			return "time.Time"
		}
		return "string"
	case float64:
		// Check if integer.
		if v == math.Trunc(v) {
			if v > -2147483648 && v < 2147483647 {
				return "int"
			}
			return "int64"
		}
		return "float64"
	case bool:
		return "bool"
	case []interface{}:
		return "slice"
	case map[string]interface{}:
		return "struct"
	default:
		return "any"
	}
}

// findBestValueForNumberType selects a better number value between existingValue and newValue.
func findBestValueForNumberType(existingValue, newValue interface{}) interface{} {
	// Check if newValue is a number.
	_, ok := newValue.(float64)
	if !ok {
		fmt.Printf("Error: currentValue %v is not a number\n", newValue)
		return nil
	}
	newGoType := goType(newValue)
	existingGoType := goType(existingValue)
	if newGoType == existingGoType {
		return existingValue
	}
	if newGoType == "float64" {
		return newValue
	}
	if existingGoType == "float64" {
		return existingValue
	}
	if strings.Contains(newGoType, "float") && strings.Contains(existingGoType, "int") {
		return math.MaxFloat64
	}
	if strings.Contains(newGoType, "int") && strings.Contains(existingGoType, "float") {
		return math.MaxFloat64
	}
	if strings.Contains(newGoType, "int") && strings.Contains(existingGoType, "int") {
		existingValueF := math.Abs(existingValue.(float64))
		newValueF := math.Abs(newValue.(float64))
		if math.IsInf(existingValueF+newValueF, 0) {
			return float64(math.MaxInt64)
		}
		return existingValueF + newValueF
	}
	fmt.Printf("Error: something went wrong with findBestValueForNumberType() using the values: '%v' and '%v'\n", newValue, existingValue)
	fmt.Println("       Please report the problem to https://github.com/mholt/json-to-go/issues")
	return nil
}

// mostSpecificPossibleGoType returns the most specific type between typ1 and typ2.
func mostSpecificPossibleGoType(typ1, typ2 string) string {
	if strings.HasPrefix(typ1, "float") && strings.HasPrefix(typ2, "int") {
		return typ1
	} else if strings.HasPrefix(typ1, "int") && strings.HasPrefix(typ2, "float") {
		return typ2
	} else {
		return "any"
	}
}

// uuidv4 generates a UUID version 4.
func uuidv4() string {
	//rand.Seed(time.Now().UnixNano())
	var uuid = []byte("xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx")
	for i, c := range uuid {
		if c == 'x' || c == 'y' {
			r := rand.Intn(16)
			if c == 'x' {
				uuid[i] = "0123456789abcdef"[r]
			} else {
				// For 'y', ensure the first hex digit is in [8, 9, a, b]
				uuid[i] = "89ab"[r%4]
			}
		}
	}
	return string(uuid)
}

// getOriginalName returns the original name by removing a trailing UUID.
func getOriginalName(unique string) string {
	reLiteralUUID := regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	uuidLength := 36
	if len(unique) >= uuidLength {
		tail := unique[len(unique)-uuidLength:]
		if reLiteralUUID.MatchString(tail) {
			return unique[:len(unique)-(uuidLength+1)]
		}
	}
	return unique
}

// areObjects checks if both objectA and objectB are plain objects.
func areObjects(objectA, objectB interface{}) bool {
	_, okA := objectA.(map[string]interface{})
	_, okB := objectB.(map[string]interface{})
	return okA && okB
}

// areSameType checks if both objects have the same type.
func areSameType(objectA, objectB interface{}) bool {
	return fmt.Sprintf("%T", objectA) == fmt.Sprintf("%T", objectB)
}

// compareObjectKeys compares two slices of keys.
func compareObjectKeys(itemAKeys, itemBKeys []string) bool {
	lengthA := len(itemAKeys)
	lengthB := len(itemBKeys)
	if lengthA == 0 && lengthB == 0 {
		return true
	}
	if lengthA != lengthB {
		return false
	}
	for _, item := range itemAKeys {
		if !containsStr(itemBKeys, item) {
			return false
		}
	}
	return true
}

// formatScopeKeys formats all keys using format.
func formatScopeKeys(keys []string) []string {
	for i, k := range keys {
		keys[i] = format(k)
	}
	return keys
}

// parseScope recursively parses the scope.
func parseScope(scope interface{}, depth int, flatten, example, allOmitempty bool) {
	if scopeMap, ok := scope.(map[string]interface{}); ok {
		// It's an object.
		if flatten {
			if depth >= 2 {
				appender(parent)
			} else {
				appendFunc(parent)
			}
		}
		parseStruct(depth+1, innerTabs, scopeMap, nil, previousParents, flatten, example, allOmitempty)
	} else if scopeArr, ok := scope.([]interface{}); ok {
		// It's an array.
		var sliceType string
		scopeLength := len(scopeArr)
		for i := 0; i < scopeLength; i++ {
			thisType := goType(scopeArr[i])
			if sliceType == "" {
				sliceType = thisType
			} else if sliceType != thisType {
				sliceType = mostSpecificPossibleGoType(thisType, sliceType)
				if sliceType == "any" {
					break
				}
			}
		}
		var sliceStr string
		if flatten && (sliceType == "struct" || sliceType == "slice") {
			sliceStr = "[]" + parent
		} else {
			sliceStr = "[]"
		}
		if flatten && depth >= 2 {
			appender(sliceStr)
		} else {
			appendFunc(sliceStr)
		}
		if sliceType == "struct" {
			allFields := make(map[string]struct {
				value interface{}
				count int
			})
			for i := 0; i < scopeLength; i++ {
				// Convert element to map.
				elem, ok := scopeArr[i].(map[string]interface{})
				if !ok {
					continue
				}
				var keys []string
				for key := range elem {
					keys = append(keys, key)
				}
				for _, keyname := range keys {
					if _, exists := allFields[keyname]; !exists {
						allFields[keyname] = struct {
							value interface{}
							count int
						}{value: elem[keyname], count: 0}
					} else {
						existing := allFields[keyname]
						existingValue := existing.value
						currentValue := elem[keyname]
						if !areSameType(existingValue, currentValue) {
							if existingValue != nil {
								existing.value = nil
								fmt.Printf("Warning: key \"%s\" uses multiple types. Defaulting to type \"any\".\n", keyname)
							}
							existing.count++
							allFields[keyname] = existing
							continue
						}
						// Check for number improvement.
						if areSameType(currentValue, float64(1)) {
							existing.value = findBestValueForNumberType(existingValue, currentValue)
						}
						if areObjects(existingValue, currentValue) {
							// Compare keys.
							keysCurrent := mapKeys(currentValue.(map[string]interface{}))
							keysExisting := mapKeys(existingValue.(map[string]interface{}))
							comparisonResult := compareObjectKeys(keysCurrent, keysExisting)
							if !comparisonResult {
								keyname = keyname + "_" + uuidv4()
								allFields[keyname] = struct {
									value interface{}
									count int
								}{value: currentValue, count: 0}
							}
						}
						existing.count++
						allFields[keyname] = existing
					}
				}
			}
			var keys []string
			structObj := make(map[string]interface{})
			omitemptyMap := make(map[string]bool)
			for key, elem := range allFields {
				keys = append(keys, key)
				structObj[key] = elem.value
				omitemptyMap[key] = (elem.count != scopeLength)
			}
			parseStruct(depth+1, innerTabs, structObj, omitemptyMap, previousParents, flatten, example, allOmitempty)
		} else if sliceType == "slice" {
			if scopeLength > 0 {
				parseScope(scopeArr[0], depth, flatten, example, allOmitempty)
			}
		} else {
			if flatten && depth >= 2 {
				appender(sliceType)
			} else {
				appendFunc(sliceType)
			}
		}
	} else {
		if flatten && depth >= 2 {
			appender(goType(scope))
		} else {
			appendFunc(goType(scope))
		}
	}
}

// parseStruct parses a struct scope.
func parseStruct(depth, innerTabs int, scope map[string]interface{}, omitempty map[string]bool, oldParents string, flatten, example, allOmitempty bool) {
	if flatten {
		if depth >= 2 {
			stack = append(stack, "\n")
		} else {
			stack = append(stack, "\n")
		}
	}
	var seenTypeNames []string
	if flatten && depth >= 2 {
		parentType := "type " + parent
		var scopeKeys []string
		for key := range scope {
			scopeKeys = append(scopeKeys, key)
		}
		scopeKeys = formatScopeKeys(scopeKeys)
		if val, exists := seen[parent]; exists {
			if compareObjectKeys(scopeKeys, val) {
				// pop from stack and return
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				return
			}
		}
		seen[parent] = scopeKeys
		appender(parentType + " struct {\n")
		innerTabs++
		var keys []string
		for key := range scope {
			keys = append(keys, key)
		}
		previousParents = parent
		for _, key := range keys {
			keyname := getOriginalName(key)
			// indenter for innerTabs
			indenter(innerTabs)
			var typenameLocal string
			if subObj, ok := scope[key]; ok {
				if asMap, ok2 := subObj.(map[string]interface{}); ok2 && asMap != nil {
					typenameLocal = uniqueTypeName(format(keyname), globallySeenTypeNames, previousParents)
					globallySeenTypeNames = append(globallySeenTypeNames, typenameLocal)
				} else {
					typenameLocal = uniqueTypeName(format(keyname), seenTypeNames)
					seenTypeNames = append(seenTypeNames, typenameLocal)
				}
			}
			appender(typenameLocal + " ")
			parent = typenameLocal
			parseScope(scope[key], depth, flatten, example, allOmitempty)
			appender(" `json:\"" + keyname)
			if allOmitempty || (omitempty != nil && omitempty[key] == true) {
				appender(",omitempty")
			}
			appender("\"`\n")
		}
		innerTabs--
		// indenter for innerTabs
		indenter(innerTabs)
		appender("}")
		previousParents = oldParents
	} else {
		appendFunc("struct {\n")
		tabs++
		var keys []string
		for key := range scope {
			keys = append(keys, key)
		}
		previousParents = parent
		for _, key := range keys {
			keyname := getOriginalName(key)
			indent(tabs)
			var typenameLocal string
			if subObj, ok := scope[key]; ok {
				if asMap, ok2 := subObj.(map[string]interface{}); ok2 && asMap != nil {
					typenameLocal = uniqueTypeName(format(keyname), globallySeenTypeNames, previousParents)
					globallySeenTypeNames = append(globallySeenTypeNames, typenameLocal)
				} else {
					typenameLocal = uniqueTypeName(format(keyname), seenTypeNames)
					seenTypeNames = append(seenTypeNames, typenameLocal)
				}
			}
			appendFunc(typenameLocal + " ")
			parent = typenameLocal
			parseScope(scope[key], depth, flatten, example, allOmitempty)
			appendFunc(" `json:\"" + keyname)
			if allOmitempty || (omitempty != nil && omitempty[key] == true) {
				appendFunc(",omitempty")
			}
			if example && scope[key] != "" {
				switch scope[key].(type) {
				case string, float64, bool:
					appendFunc("\" example:\"" + fmt.Sprintf("%v", scope[key]))
				default:
					// do nothing for objects
				}
			}
			appendFunc("\"`\n")
		}
		tabs--
		indent(tabs)
		appendFunc("}")
		previousParents = oldParents
	}
	if flatten {
		if len(stack) > 0 {
			accumulator += stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		}
	}
}

// Helper: returns the keys of a map.
func mapKeys(m map[string]interface{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Local helper functions defined as closures
// Helper: checks if a slice contains a string.
func contains(slice []string, s string) bool {
	return containsStr(slice, s)
}

func jsonToGo(jsonStr, typename string, flatten, example, allOmitempty bool) JsonToGoOutput {
	var data interface{}
	var scope interface{}
	goResult = ""
	accumulator = ""

	tabs = 0
	seen = make(map[string][]string)
	innerTabs = 0
	parent = ""
	previousParents = ""

	// Begin main logic of jsonToGo.
	// Replace regex (:\\s*\\[?\\s*-?\\d*)\\.0 with $1.1
	re := regexp.MustCompile(`(:\s*\[?\s*-?\d*)\.0`)
	jsonModified := re.ReplaceAllString(jsonStr, "$1.1")
	err := json.Unmarshal([]byte(jsonModified), &data)
	if err != nil {
		return JsonToGoOutput{Go: "", Error: err.Error()}
	}
	scope = data
	if typename == "" {
		typename = "AutoGenerated"
	}
	typename = format(typename)
	appendFunc("type " + typename + " ")
	parseScope(scope, 0, flatten, example, allOmitempty)
	if flatten {
		goResult += accumulator
	}
	if !strings.HasSuffix(goResult, "\n") {
		goResult += "\n"
	}
	return JsonToGoOutput{Go: goResult}
}

func main() {
	// If run as command line tool.
	var filename string
	// Process command line arguments.
	for index, val := range os.Args {
		if index < 1 {
			continue
		}
		if !strings.HasPrefix(val, "-") {
			filename = val
			continue
		}
		argument := strings.Replace(val, "-", "", -1)
		if argument == "big" {
			log.Fatal(fmt.Printf("Warning: The argument '%s' has been deprecated and has no effect anymore\n", argument))
		} else {
			log.Fatal(fmt.Printf("Unexpected argument %s received\n", val))
		}
	}
	var input string

	// Read from file if filename is provided.
	if filename != "" {
		buff, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(fmt.Println(err))
		}
		input = string(buff)
	} else {
		// Otherwise, read from stdin.
		buff := bytes.Buffer{}
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			buff.WriteString(scanner.Text())
		}
		input = buff.String()
	}
	// convert json to go struct
	output := jsonToGo(input, "", true, false, false)
	if output.Error != "" {
		log.Fatal(fmt.Println(output.Error))
	}
	// go fmt the generated go struct
	formatted, err := goformat.Source([]byte(output.Go))
	if err != nil {
		log.Fatal(err)
	}
	// print the result on stdout
	fmt.Print(string(formatted))
}
