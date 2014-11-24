package flotilla

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	regDoubleQuote = regexp.MustCompile("^([^= \t]+)[ \t]*=[ \t]*\"([^\"]*)\"$")
	regSingleQuote = regexp.MustCompile("^([^= \t]+)[ \t]*=[ \t]*'([^']*)'$")
	regNoQuote     = regexp.MustCompile("^([^= \t]+)[ \t]*=[ \t]*([^#;]+)")
	regNoValue     = regexp.MustCompile("^([^= \t]+)[ \t]*=[ \t]*([#;].*)?")

	boolString = map[string]bool{
		"t":     true,
		"true":  true,
		"y":     true,
		"yes":   true,
		"on":    true,
		"1":     true,
		"f":     false,
		"false": false,
		"n":     false,
		"no":    false,
		"off":   false,
		"0":     false,
	}
)

type (
	// A StoreItem contains a default string value and/or a string value.
	StoreItem struct {
		defaultvalue bool
		Value        string
	}

	// Store is a map of StoreItem managed by App.Env, used as a store of varied
	// configuration items that might be represented with a default and/or explicitly
	// set value.
	Store map[string]*StoreItem
)

// LoadConfFile loads a text configuration file into a Store.
func (s Store) LoadConfFile(filename string) (err error) {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	err = s.parse(reader, filename)
	return err
}

// LoadConfByte loads a text configuration file as byte into a Store.
func (s Store) LoadConfByte(b []byte, name string) (err error) {
	reader := bufio.NewReader(bytes.NewReader(b))
	err = s.parse(reader, name)
	return err
}

func (s Store) parse(reader *bufio.Reader, filename string) (err error) {
	lineno := 0
	section := ""
	for err == nil {
		l, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		lineno++
		if len(l) == 0 {
			continue
		}
		line := strings.TrimFunc(string(l), unicode.IsSpace)
		for line[len(line)-1] == '\\' {
			line = line[:len(line)-1]
			l, _, err := reader.ReadLine()
			if err != nil {
				return err
			}
			line += strings.TrimFunc(string(l), unicode.IsSpace)
		}
		section, err = s.parseLine(section, line)
		if err != nil {
			return newError("[FLOTILLA] configuration parser: syntax error at '%s:%d'.", filename, lineno)
		}
	}
	return err
}

func (s Store) parseLine(section, line string) (string, error) {
	if line[0] == '#' || line[0] == ';' {
		return section, nil
	}

	if line[0] == '[' && line[len(line)-1] == ']' {
		section := strings.TrimFunc(line[1:len(line)-1], unicode.IsSpace)
		section = strings.ToLower(section)
		return section, nil
	}

	if m := regDoubleQuote.FindAllStringSubmatch(line, 1); m != nil {
		s.add(section, m[0][1], m[0][2])
		return section, nil
	} else if m = regSingleQuote.FindAllStringSubmatch(line, 1); m != nil {
		s.add(section, m[0][1], m[0][2])
		return section, nil
	} else if m = regNoQuote.FindAllStringSubmatch(line, 1); m != nil {
		s.add(section, m[0][1], strings.TrimFunc(m[0][2], unicode.IsSpace))
		return section, nil
	} else if m = regNoValue.FindAllStringSubmatch(line, 1); m != nil {
		s.add(section, m[0][1], "")
		return section, nil
	}
	return section, newError("flotilla env conf parse error")
}

func (s Store) newkey(section string, key string) string {
	if len(section) != 0 {
		key = fmt.Sprintf("%s_%s", section, strings.ToLower(key))
	}
	return strings.ToUpper(key)
}

func (s Store) add(section, key, value string) {
	s[s.newkey(section, key)] = &StoreItem{Value: value, defaultvalue: false}
}

func (s Store) adddefault(section, key, value string) {
	s[s.newkey(section, key)] = &StoreItem{Value: value, defaultvalue: true}
}

// Bool attempts to return the storeitem value as type bool
func (si StoreItem) Bool() (bool, error) {
	if value, ok := boolString[strings.ToLower(si.Value)]; ok {
		return value, nil
	}
	return false, newError("could not return Bool value from StoreItem")
}

// Float attempts to return the storeitem value as type float
func (si *StoreItem) Float() (value float64, err error) {
	if value, err := strconv.ParseFloat(si.Value, 64); err == nil {
		return value, nil
	}
	return 0.0, newError("could not return Float value from StoreItem")
}

// Int attempts to return the storeitem value as type int
func (si *StoreItem) Int() (value int, err error) {
	if value, err := strconv.Atoi(si.Value); err == nil {
		return value, nil
	}
	return 0, newError("could not return Int value from StoreItem")
}

// Int64 attempts to return the storeitem value as type int64
func (si *StoreItem) Int64() (value int64, err error) {
	if value, err := strconv.ParseInt(si.Value, 10, 64); err == nil {
		return value, nil
	}
	return 0, newError("could not return Int64 value from StoreItem")
}

// List updates the storeitem value to a list with the provided strings, and
// then returns the updated value as a string array type.
func (si *StoreItem) List(li ...string) []string {
	list := strings.Split(si.Value, ",")
	for _, item := range li {
		list = doAdd(item, list)
	}
	si.Value = strings.Join(list, ",")
	return strings.Split(si.Value, ",")
}
