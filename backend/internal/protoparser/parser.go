package protoparser

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidProto = errors.New("invalid proto")

	serviceRe = regexp.MustCompile(`(?s)service\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{(.*?)\}`)
	rpcRe     = regexp.MustCompile(`rpc\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*([^)]+?)\s*\)\s+returns\s+\(\s*([^)]+?)\s*\)`)
	msgRe     = regexp.MustCompile(`(?s)message\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{(.*?)\}`)
	fieldRe   = regexp.MustCompile(`(?m)^\s*(?:repeated\s+)?([A-Za-z_][A-Za-z0-9_\.<>]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\d+\s*;`)
)

type Contract struct {
	ServiceName string
	Methods     []Method
}

type Method struct {
	Name       string
	Input      string
	Output     string
	Parameters []Parameter
}

type Parameter struct {
	Name string
	Type string
}

func ParseProjectProto(src []byte) (Contract, error) {
	text := stripComments(string(src))
	svc := serviceRe.FindStringSubmatch(text)
	if len(svc) < 3 {
		return Contract{}, ErrInvalidProto
	}

	msgMap := parseMessages(text)
	methodMatches := rpcRe.FindAllStringSubmatch(svc[2], -1)
	if len(methodMatches) == 0 {
		return Contract{}, ErrInvalidProto
	}

	out := Contract{
		ServiceName: strings.TrimSpace(svc[1]),
		Methods:     make([]Method, 0, len(methodMatches)),
	}
	for _, m := range methodMatches {
		in := cleanType(m[2])
		method := Method{
			Name:   strings.TrimSpace(m[1]),
			Input:  in,
			Output: cleanType(m[3]),
		}
		if params, ok := msgMap[in]; ok {
			method.Parameters = params
		}
		out.Methods = append(out.Methods, method)
	}

	return out, nil
}

func parseMessages(text string) map[string][]Parameter {
	matches := msgRe.FindAllStringSubmatch(text, -1)
	out := make(map[string][]Parameter, len(matches))
	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		body := m[2]
		fields := fieldRe.FindAllStringSubmatch(body, -1)
		params := make([]Parameter, 0, len(fields))
		for _, f := range fields {
			params = append(params, Parameter{
				Type: strings.TrimSpace(f[1]),
				Name: strings.TrimSpace(f[2]),
			})
		}
		out[name] = params
	}
	return out
}

func cleanType(t string) string {
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(t, "stream ")
	return strings.TrimSpace(t)
}

func stripComments(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}

