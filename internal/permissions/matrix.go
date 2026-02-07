package permissions

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"
)

// Matrix represents a permission access matrix.
type Matrix struct {
	Services []string
	Methods  map[string][]string // service -> methods
	Roles    []string
	Scopes   []string
	Access   map[string]map[string]*AccessLevel // "service.method" -> role/scope -> access
}

// BuildMatrix builds a permission matrix from services.
func BuildMatrix(services []*ServicePermissions) *Matrix {
	m := &Matrix{
		Methods: make(map[string][]string),
		Access:  make(map[string]map[string]*AccessLevel),
	}

	roleSet := make(map[string]bool)
	scopeSet := make(map[string]bool)

	for _, svc := range services {
		m.Services = append(m.Services, svc.Service)

		for method, perm := range svc.Methods {
			m.Methods[svc.Service] = append(m.Methods[svc.Service], method)

			key := svc.Service + "." + method
			m.Access[key] = make(map[string]*AccessLevel)

			// Track roles
			for _, role := range perm.Roles {
				roleSet[role] = true
				m.Access[key]["role:"+role] = &AccessLevel{
					Allowed:    true,
					RequireMFA: perm.RequireMFA,
					Conditions: len(perm.Conditions) > 0,
					RateLimit:  perm.RateLimit != nil,
				}
			}

			// Track scopes
			for _, scope := range perm.Scopes {
				scopeSet[scope] = true
				m.Access[key]["scope:"+scope] = &AccessLevel{
					Allowed:    true,
					RequireMFA: perm.RequireMFA,
					Conditions: len(perm.Conditions) > 0,
					RateLimit:  perm.RateLimit != nil,
				}
			}

			// Mark public endpoints
			if perm.Public {
				m.Access[key]["public"] = &AccessLevel{
					Allowed:   true,
					RateLimit: perm.RateLimit != nil,
				}
			}
		}
	}

	// Sort roles and scopes
	for role := range roleSet {
		m.Roles = append(m.Roles, role)
	}
	sort.Strings(m.Roles)

	for scope := range scopeSet {
		m.Scopes = append(m.Scopes, scope)
	}
	sort.Strings(m.Scopes)

	// Sort methods per service
	for svc := range m.Methods {
		sort.Strings(m.Methods[svc])
	}

	sort.Strings(m.Services)

	return m
}

// RenderText renders the matrix as text table.
func (m *Matrix) RenderText() string {
	var buf bytes.Buffer

	// Calculate column widths
	methodColWidth := 30
	roleColWidth := 12

	// Header
	buf.WriteString(strings.Repeat("=", methodColWidth+len(m.Roles)*roleColWidth+len(m.Scopes)*roleColWidth+10))
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("%-*s", methodColWidth, "Method"))
	for _, role := range m.Roles {
		buf.WriteString(fmt.Sprintf("%-*s", roleColWidth, truncate(role, roleColWidth-1)))
	}
	for _, scope := range m.Scopes {
		buf.WriteString(fmt.Sprintf("%-*s", roleColWidth, truncate("s:"+scope, roleColWidth-1)))
	}
	buf.WriteString("Public")
	buf.WriteString("\n")
	buf.WriteString(strings.Repeat("-", methodColWidth+len(m.Roles)*roleColWidth+len(m.Scopes)*roleColWidth+10))
	buf.WriteString("\n")

	// Data rows
	for _, svc := range m.Services {
		// Service header
		buf.WriteString(fmt.Sprintf("\n[%s]\n", svc))

		methods := m.Methods[svc]
		for _, method := range methods {
			key := svc + "." + method
			buf.WriteString(fmt.Sprintf("%-*s", methodColWidth, truncate(method, methodColWidth-1)))

			// Role columns
			for _, role := range m.Roles {
				access := m.Access[key]["role:"+role]
				buf.WriteString(fmt.Sprintf("%-*s", roleColWidth, formatAccess(access)))
			}

			// Scope columns
			for _, scope := range m.Scopes {
				access := m.Access[key]["scope:"+scope]
				buf.WriteString(fmt.Sprintf("%-*s", roleColWidth, formatAccess(access)))
			}

			// Public column
			access := m.Access[key]["public"]
			buf.WriteString(formatAccess(access))
			buf.WriteString("\n")
		}
	}

	buf.WriteString("\n")
	buf.WriteString("Legend: ✓ = allowed, M = requires MFA, C = has conditions, R = rate limited\n")

	return buf.String()
}

// RenderHTML renders the matrix as HTML table.
func (m *Matrix) RenderHTML() (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Permission Matrix</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 20px; }
        h1 { color: #333; }
        h2 { color: #666; margin-top: 30px; border-bottom: 2px solid #eee; padding-bottom: 5px; }
        table { border-collapse: collapse; width: 100%; margin-bottom: 30px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: center; }
        th { background-color: #f5f5f5; font-weight: 600; }
        th.method { text-align: left; min-width: 200px; }
        td.method { text-align: left; font-family: monospace; }
        .allowed { background-color: #d4edda; color: #155724; }
        .denied { background-color: #f8f9fa; color: #999; }
        .mfa { background-color: #fff3cd; color: #856404; }
        .legend { margin-top: 20px; padding: 15px; background: #f8f9fa; border-radius: 4px; }
        .legend span { margin-right: 20px; }
        .role-header { background-color: #e3f2fd; }
        .scope-header { background-color: #f3e5f5; }
        .public-header { background-color: #fff8e1; }
    </style>
</head>
<body>
    <h1>Permission Access Matrix</h1>
    
    {{range .Services}}
    <h2>{{.}}</h2>
    <table>
        <thead>
            <tr>
                <th class="method">Method</th>
                {{range $.Roles}}<th class="role-header">{{.}}</th>{{end}}
                {{range $.Scopes}}<th class="scope-header">{{.}}</th>{{end}}
                <th class="public-header">Public</th>
            </tr>
        </thead>
        <tbody>
            {{$svc := .}}
            {{range index $.Methods .}}
            <tr>
                <td class="method">{{.}}</td>
                {{$key := printf "%s.%s" $svc .}}
                {{range $.Roles}}
                    {{$access := index (index $.Access $key) (printf "role:%s" .)}}
                    <td class="{{accessClass $access}}">{{accessSymbol $access}}</td>
                {{end}}
                {{range $.Scopes}}
                    {{$access := index (index $.Access $key) (printf "scope:%s" .)}}
                    <td class="{{accessClass $access}}">{{accessSymbol $access}}</td>
                {{end}}
                {{$public := index (index $.Access $key) "public"}}
                <td class="{{accessClass $public}}">{{accessSymbol $public}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    <div class="legend">
        <strong>Legend:</strong>
        <span>✓ = Allowed</span>
        <span>✓ᴹ = Requires MFA</span>
        <span>✓ᶜ = Has Conditions</span>
        <span>✓ʳ = Rate Limited</span>
        <span>— = Not Allowed</span>
    </div>
</body>
</html>`

	funcMap := template.FuncMap{
		"accessClass": func(a *AccessLevel) string {
			if a == nil || !a.Allowed {
				return "denied"
			}
			if a.RequireMFA {
				return "mfa"
			}
			return "allowed"
		},
		"accessSymbol": func(a *AccessLevel) string {
			if a == nil || !a.Allowed {
				return "—"
			}
			sym := "✓"
			if a.RequireMFA {
				sym += "ᴹ"
			}
			if a.Conditions {
				sym += "ᶜ"
			}
			if a.RateLimit {
				sym += "ʳ"
			}
			return sym
		},
	}

	t, err := template.New("matrix").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, m); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// RenderMarkdown renders the matrix as Markdown table.
func (m *Matrix) RenderMarkdown() string {
	var buf bytes.Buffer

	buf.WriteString("# Permission Access Matrix\n\n")

	for _, svc := range m.Services {
		buf.WriteString(fmt.Sprintf("## %s\n\n", svc))

		// Header row
		buf.WriteString("| Method |")
		for _, role := range m.Roles {
			buf.WriteString(fmt.Sprintf(" %s |", role))
		}
		for _, scope := range m.Scopes {
			buf.WriteString(fmt.Sprintf(" %s |", scope))
		}
		buf.WriteString(" Public |\n")

		// Separator row
		buf.WriteString("|--------|")
		for range m.Roles {
			buf.WriteString(":------:|")
		}
		for range m.Scopes {
			buf.WriteString(":------:|")
		}
		buf.WriteString(":------:|\n")

		// Data rows
		methods := m.Methods[svc]
		for _, method := range methods {
			key := svc + "." + method
			buf.WriteString(fmt.Sprintf("| `%s` |", method))

			for _, role := range m.Roles {
				access := m.Access[key]["role:"+role]
				buf.WriteString(fmt.Sprintf(" %s |", formatAccessMD(access)))
			}

			for _, scope := range m.Scopes {
				access := m.Access[key]["scope:"+scope]
				buf.WriteString(fmt.Sprintf(" %s |", formatAccessMD(access)))
			}

			public := m.Access[key]["public"]
			buf.WriteString(fmt.Sprintf(" %s |\n", formatAccessMD(public)))
		}

		buf.WriteString("\n")
	}

	buf.WriteString("### Legend\n\n")
	buf.WriteString("- ✓ = Allowed\n")
	buf.WriteString("- ✓ᴹ = Requires MFA\n")
	buf.WriteString("- ✓ᶜ = Has Conditions\n")
	buf.WriteString("- ✓ʳ = Rate Limited\n")
	buf.WriteString("- — = Not Allowed\n")

	return buf.String()
}

func formatAccess(a *AccessLevel) string {
	if a == nil || !a.Allowed {
		return "—"
	}
	sym := "✓"
	if a.RequireMFA {
		sym += "M"
	}
	if a.Conditions {
		sym += "C"
	}
	if a.RateLimit {
		sym += "R"
	}
	return sym
}

func formatAccessMD(a *AccessLevel) string {
	if a == nil || !a.Allowed {
		return "—"
	}
	sym := "✓"
	if a.RequireMFA {
		sym += "ᴹ"
	}
	if a.Conditions {
		sym += "ᶜ"
	}
	if a.RateLimit {
		sym += "ʳ"
	}
	return sym
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
