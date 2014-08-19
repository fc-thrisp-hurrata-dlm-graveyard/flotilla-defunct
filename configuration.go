package fleet

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

var (
	regDoubleQuote = regexp.MustCompile("^([^= \t]+)[ \t]*=[ \t]*\"([^\"]*)\"$")
	regSingleQuote = regexp.MustCompile("^([^= \t]+)[ \t]*=[ \t]*'([^']*)'$")
	regNoQuote     = regexp.MustCompile("^([^= \t]+)[ \t]*=[ \t]*([^#;]+)")
	regNoValue     = regexp.MustCompile("^([^= \t]+)[ \t]*=[ \t]*([#;].*)?")
)

type (
	FleetConf map[string]string
)

func (env *FleetEnv) getConf() FleetConf {
	if env.FleetConf == nil {
		return make(map[string]string)
	} else {
		return env.FleetConf
	}
}

func (env *FleetEnv) LoadConfFile(filename string) (FleetConf, error) {
	f := env.getConf()
	file, err := os.Open(filename)
	if err != nil {
		return f, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	f, err = parse(f, reader, filename)
	return f, err
}

func (env *FleetEnv) LoadConfByte(b []byte, name string) (returnedf FleetConf, err error) {
	f := env.getConf()
	reader := bufio.NewReader(bytes.NewReader(b))
	f, err = parse(f, reader, name)
	return f, err
}

func parse(f FleetConf, reader *bufio.Reader, filename string) (returnedf FleetConf, err error) {
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
				return nil, err
			}
			line += strings.TrimFunc(string(l), unicode.IsSpace)
		}
		section, err = f.parseLine(section, line)
		if err != nil {
			return nil, newError(
				err.Error() + fmt.Sprintf("'%s:%d'.", filename, lineno))
		}
	}
	return f, err
}

func (f FleetConf) parseLine(section, line string) (string, error) {
	if line[0] == '#' || line[0] == ';' {
		return section, nil
	}

	if line[0] == '[' && line[len(line)-1] == ']' {
		section := strings.TrimFunc(line[1:len(line)-1], unicode.IsSpace)
		section = strings.ToLower(section)
		return section, nil
	}

	if m := regDoubleQuote.FindAllStringSubmatch(line, 1); m != nil {
		f.add(section, m[0][1], m[0][2])
		return section, nil
	} else if m = regSingleQuote.FindAllStringSubmatch(line, 1); m != nil {
		f.add(section, m[0][1], m[0][2])
		return section, nil
	} else if m = regNoQuote.FindAllStringSubmatch(line, 1); m != nil {
		f.add(section, m[0][1], strings.TrimFunc(m[0][2], unicode.IsSpace))
		return section, nil
	} else if m = regNoValue.FindAllStringSubmatch(line, 1); m != nil {
		f.add(section, m[0][1], "")
		return section, nil
	}
	return section, newError("iniparser: syntax error at ")
}

func (f FleetConf) add(section, key, value string) {
	if len(section) != 0 {
		key = fmt.Sprintf("%s_%s", section, strings.ToLower(key))
	}
	f[strings.ToLower(key)] = value
}
