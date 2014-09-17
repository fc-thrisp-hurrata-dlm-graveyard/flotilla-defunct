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
	StoreItem struct {
		defaultvalue bool
		value        string
	}

	Store map[string]*StoreItem
)

// Loads a text configuration file into the store
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

// Loads a text configuration file as byte into the store
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
			return newError("flotilla configuration parser: syntax error at '%s:%d'.", filename, lineno)
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
	s[s.newkey(section, key)] = &StoreItem{value: value, defaultvalue: false}
}

func (s Store) adddefault(section, key, value string) {
	s[s.newkey(section, key)] = &StoreItem{value: value, defaultvalue: true}
}

func (si StoreItem) Bool() (bool, error) {
	if value, ok := boolString[strings.ToLower(si.value)]; ok {
		return value, nil
	}
	return false, newError("could not return Bool value from StoreItem value")
}

func (si *StoreItem) Float() (value float64, err error) {
	if value, err := strconv.ParseFloat(si.value, 64); err == nil {
		return value, nil
	}
	return 0.0, newError("could not return Float value from StoreItem value")
}

func (si *StoreItem) Int() (value int, err error) {
	if value, err := strconv.Atoi(si.value); err == nil {
		return value, nil
	}
	return 0, newError("could not return Int value from StoreItem value")
}

func (si *StoreItem) List() (value []string, err error) {
	return strings.Split(si.value, ","), nil
}

func (si *StoreItem) update(s string) {
	si.value = s
}

func (si *StoreItem) updateList(li ...string) {
	if list, err := si.List(); err == nil {
		for _, item := range li {
			list = dirAdd(item, list)
		}
		si.value = strings.Join(list, ",")
	}
}
