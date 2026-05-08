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
	fieldRe   = regexp.MustCompile(`(?m)^\s*(repeated\s+)?([A-Za-z_][A-Za-z0-9_\.<>]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\d+\s*;`)
)

type Contract struct {
	ServiceName string   `json:"service_name"`
	Methods     []Method `json:"methods"`
}

type Method struct {
	Name   string      `json:"name"`
	Input  MessageSpec `json:"input"`
	Output MessageSpec `json:"output"`
}

type MessageSpec struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
}

type Parameter struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Repeated bool        `json:"repeated"`
	Children []Parameter `json:"children,omitempty"`
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
		inputName := cleanType(m[2])
		outputName := cleanType(m[3])

		method := Method{
			Name:   strings.TrimSpace(m[1]),
			Input:  buildMessageSpec(inputName, msgMap, nil),
			Output: buildMessageSpec(outputName, msgMap, nil),
		}

		out.Methods = append(out.Methods, method)
	}

	return out, nil
}

func buildMessageSpec(name string, msgMap map[string][]Parameter, visited map[string]bool) MessageSpec {
	if visited == nil {
		visited = make(map[string]bool)
	}

	return MessageSpec{
		Name:       name,
		Parameters: expandParameters(name, msgMap, visited),
	}
}

func expandParameters(messageName string, msgMap map[string][]Parameter, visited map[string]bool) []Parameter {
	params, ok := msgMap[messageName]
	if !ok {
		return nil
	}

	if visited[messageName] {
		return params
	}

	visited[messageName] = true
	defer delete(visited, messageName)

	out := make([]Parameter, 0, len(params))
	for _, p := range params {
		cp := p

		if nested, ok := msgMap[p.Type]; ok && len(nested) > 0 {
			cp.Children = expandParameters(p.Type, msgMap, visited)
		}

		out = append(out, cp)
	}

	return out
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
				Repeated: strings.TrimSpace(f[1]) != "",
				Type:     strings.TrimSpace(f[2]),
				Name:     strings.TrimSpace(f[3]),
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