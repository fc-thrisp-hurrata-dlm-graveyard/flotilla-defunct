package flotilla

var (
	builtintplfuncs = map[string]interface{}{
		"getflashmessages": getflashmessages,
	}
)

func getflashmessages(t *TData, categories ...string) []string {
	var ret []string
	for k, v := range t.Flash {
		if existsIn(k, categories) {
			ret = append(ret, v)
		}
	}
	return ret
}
