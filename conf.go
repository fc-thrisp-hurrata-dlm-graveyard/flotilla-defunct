package flotilla

import (
	"bufio"
	"fmt"
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
	boolString     = map[string]bool{
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
	Conf map[string]string
)

func (c Conf) parsemap(m Conf) (err error) {
	for k, v := range m {
		c.add("", k, v)
	}
	return err
}

func (c Conf) parse(reader *bufio.Reader, filename string) (err error) {
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
		section, err = c.parseLine(section, line)
		if err != nil {
			return newError("flotilla conf parser: syntax error at '%s:%d'.", filename, lineno)
		}
	}
	return err
}

func (c Conf) parseLine(section, line string) (string, error) {
	if line[0] == '#' || line[0] == ';' {
		return section, nil
	}

	if line[0] == '[' && line[len(line)-1] == ']' {
		section := strings.TrimFunc(line[1:len(line)-1], unicode.IsSpace)
		section = strings.ToLower(section)
		return section, nil
	}

	if m := regDoubleQuote.FindAllStringSubmatch(line, 1); m != nil {
		c.add(section, m[0][1], m[0][2])
		return section, nil
	} else if m = regSingleQuote.FindAllStringSubmatch(line, 1); m != nil {
		c.add(section, m[0][1], m[0][2])
		return section, nil
	} else if m = regNoQuote.FindAllStringSubmatch(line, 1); m != nil {
		c.add(section, m[0][1], strings.TrimFunc(m[0][2], unicode.IsSpace))
		return section, nil
	} else if m = regNoValue.FindAllStringSubmatch(line, 1); m != nil {
		c.add(section, m[0][1], "")
		return section, nil
	}
	return section, newError("flotilla conf parse error")
}

func (c Conf) newkey(section string, key string) string {
	if len(section) != 0 {
		key = fmt.Sprintf("%s_%s", section, strings.ToLower(key))
	}
	return strings.ToLower(key)
}

func (c Conf) add(section, key, value string) {
	c[c.newkey(section, key)] = value
}

func (c Conf) rawstring(key string) (string, error) {
	if value, ok := c[strings.ToLower(key)]; ok {
		return value, nil
	}
	return "", newError("key: %s is unavailable or does not exist", key)
}

func (c Conf) Bool(key string) (bool, error) {
	if val, err := c.rawstring(key); err == nil {
		if value, ok := boolString[strings.ToLower(val)]; ok {
			return value, nil
		}
	}
	return false, newError("could not parse bool value from key: %s", key)
}

func (c Conf) Float(key string) (value float64, err error) {
	if val, err := c.rawstring(key); err == nil {
		if value, err := strconv.ParseFloat(val, 64); err == nil {
			return value, nil
		}
	}
	return 0.0, newError("could not parse float value from key: %s -- %s", key, err)
}

func (c Conf) Int(key string) (value int, err error) {
	if val, err := c.rawstring(key); err == nil {
		if value, err := strconv.Atoi(val); err == nil {
			return value, nil
		}
	}
	return 0, newError("could not parse int value from key: %s -- %s", key, err)
}

func (c Conf) List(key string) (value []string, err error) {
	if val, err := c.rawstring(key); err == nil {
		return strings.Split(val, ","), err
	}
	return value, newError("could not parse list value from key: %s", key)
}
