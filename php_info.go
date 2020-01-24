package main

import (
	"bufio"
	"io"
	"strings"
)

type phpInfo map[string]section
type object map[string]interface{}
type section []object

func parsePHPInfo(src io.Reader) (info phpInfo) {
	info = make(phpInfo)
	r := bufio.NewReader(src)
	name := ""
	var sec section
	for {
		v, err := r.ReadString('\n')
		if err != nil {
			return
		}
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if strings.Index(v, " ") < 1 {
			// we are in a new section now. Commit previous section before moving on
			if name != "" && len(sec) > 0 {
				info[name] = sec
				name = ""
				sec = nil
			}
			name = v
			continue
		}
		if strings.Index(v, "=>") != -1 {
			parts := strings.Split(v, "=>")
			for i := 0; i < len(parts); i++ {
				parts[i] = strings.TrimSpace(parts[i])
			}
			o := make(object)
			o[parts[0]] = parts[1:]
			sec = append(sec, o)
		}
	}
	return
}
