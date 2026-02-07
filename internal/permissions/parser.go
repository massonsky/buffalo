package permissions

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/massonsky/buffalo/pkg/errors"
)

// Parser parses permission annotations from proto files.
type Parser struct {
	services map[string]*ServicePermissions
}

// NewParser creates a new permission parser.
func NewParser() *Parser {
	return &Parser{
		services: make(map[string]*ServicePermissions),
	}
}

// ParseFile parses permissions from a proto file.
func (p *Parser) ParseFile(ctx context.Context, path string) ([]*ServicePermissions, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to open proto file")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	return p.parseScanner(scanner, path)
}

// ParseDirectory parses all proto files in a directory.
func (p *Parser) ParseDirectory(ctx context.Context, dir string) ([]*ServicePermissions, error) {
	var allPermissions []*ServicePermissions

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}

		perms, err := p.ParseFile(ctx, path)
		if err != nil {
			return err
		}

		allPermissions = append(allPermissions, perms...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return allPermissions, nil
}

func (p *Parser) parseScanner(scanner *bufio.Scanner, filename string) ([]*ServicePermissions, error) {
	var services []*ServicePermissions
	var currentService *ServicePermissions
	var currentMethod string
	var inService bool
	var braceCount int
	var packageName string

	// Regex patterns
	packageRe := regexp.MustCompile(`^\s*package\s+([a-zA-Z0-9_.]+)\s*;`)
	serviceRe := regexp.MustCompile(`^\s*service\s+(\w+)\s*\{?`)
	serviceOptRe := regexp.MustCompile(`option\s*\(buffalo\.permissions\.resource\)\s*=\s*"([^"]+)"`)
	rpcRe := regexp.MustCompile(`^\s*rpc\s+(\w+)\s*\(`)
	permStartRe := regexp.MustCompile(`\[\s*\(buffalo\.permissions\)\s*=\s*\{`)
	actionRe := regexp.MustCompile(`action\s*:\s*"([^"]+)"`)
	rolesRe := regexp.MustCompile(`roles\s*:\s*\[([^\]]+)\]`)
	scopesRe := regexp.MustCompile(`scopes\s*:\s*\[([^\]]+)\]`)
	allowSelfRe := regexp.MustCompile(`allow_self\s*:\s*true`)
	requireMFARe := regexp.MustCompile(`require_mfa\s*:\s*true`)
	auditLogRe := regexp.MustCompile(`audit_log\s*:\s*true`)
	publicRe := regexp.MustCompile(`public\s*:\s*true`)

	var permBuffer strings.Builder
	var inPermission bool

	for scanner.Scan() {
		line := scanner.Text()

		// Track package
		if matches := packageRe.FindStringSubmatch(line); len(matches) > 1 {
			packageName = matches[1]
			continue
		}

		// Track service start
		if matches := serviceRe.FindStringSubmatch(line); len(matches) > 1 {
			serviceName := matches[1]
			if packageName != "" {
				serviceName = packageName + "." + serviceName
			}

			currentService = &ServicePermissions{
				Service: serviceName,
				Methods: make(map[string]*Permission),
			}
			services = append(services, currentService)
			inService = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			continue
		}

		if !inService {
			continue
		}

		// Track braces
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
		if braceCount <= 0 {
			inService = false
			currentService = nil
			continue
		}

		// Check for service resource option
		if matches := serviceOptRe.FindStringSubmatch(line); len(matches) > 1 {
			currentService.Resource = matches[1]
			continue
		}

		// Track RPC method
		if matches := rpcRe.FindStringSubmatch(line); len(matches) > 1 {
			currentMethod = matches[1]
			continue
		}

		// Track permission annotation start
		if permStartRe.MatchString(line) {
			inPermission = true
			permBuffer.Reset()
			permBuffer.WriteString(line)
			continue
		}

		if inPermission {
			permBuffer.WriteString(line)

			// Check if permission block ended
			if strings.Contains(line, "}]") || strings.Contains(line, "} ]") {
				inPermission = false
				permText := permBuffer.String()

				// Parse the permission
				perm := &Permission{}

				if matches := actionRe.FindStringSubmatch(permText); len(matches) > 1 {
					perm.Action = matches[1]
				}

				if matches := rolesRe.FindStringSubmatch(permText); len(matches) > 1 {
					perm.Roles = parseStringList(matches[1])
				}

				if matches := scopesRe.FindStringSubmatch(permText); len(matches) > 1 {
					perm.Scopes = parseStringList(matches[1])
				}

				if allowSelfRe.MatchString(permText) {
					perm.AllowSelf = true
				}

				if requireMFARe.MatchString(permText) {
					perm.RequireMFA = true
				}

				if auditLogRe.MatchString(permText) {
					perm.AuditLog = true
				}

				if publicRe.MatchString(permText) {
					perm.Public = true
				}

				// Parse conditions
				perm.Conditions = parseConditions(permText)

				// Parse rate limit
				perm.RateLimit = parseRateLimit(permText)

				if currentMethod != "" && currentService != nil {
					currentService.Methods[currentMethod] = perm
				}

				currentMethod = ""
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "error reading proto file")
	}

	return services, nil
}

func parseStringList(s string) []string {
	var result []string
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func parseConditions(text string) []Condition {
	var conditions []Condition

	// Match conditions array
	condRe := regexp.MustCompile(`conditions\s*:\s*\[(.*?)\]`)
	matches := condRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return conditions
	}

	// Parse individual conditions
	condItemRe := regexp.MustCompile(`\{([^}]+)\}`)
	items := condItemRe.FindAllStringSubmatch(matches[1], -1)

	for _, item := range items {
		if len(item) < 2 {
			continue
		}

		cond := Condition{}
		content := item[1]

		// Parse field
		if fieldMatch := regexp.MustCompile(`field\s*:\s*"([^"]+)"`).FindStringSubmatch(content); len(fieldMatch) > 1 {
			cond.Field = fieldMatch[1]
		}

		// Parse operator
		if opMatch := regexp.MustCompile(`operator\s*:\s*"([^"]+)"`).FindStringSubmatch(content); len(opMatch) > 1 {
			cond.Operator = ConditionOperator(opMatch[1])
		}

		// Parse source
		if srcMatch := regexp.MustCompile(`source\s*:\s*"([^"]+)"`).FindStringSubmatch(content); len(srcMatch) > 1 {
			cond.Source = srcMatch[1]
		}

		if cond.Field != "" {
			conditions = append(conditions, cond)
		}
	}

	return conditions
}

func parseRateLimit(text string) *RateLimit {
	rlRe := regexp.MustCompile(`rate_limit\s*:\s*\{([^}]+)\}`)
	matches := rlRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil
	}

	content := matches[1]
	rl := &RateLimit{}

	// Parse requests
	if reqMatch := regexp.MustCompile(`requests\s*:\s*(\d+)`).FindStringSubmatch(content); len(reqMatch) > 1 {
		var requests int
		if _, err := stringToInt(reqMatch[1], &requests); err == nil {
			rl.Requests = requests
		}
	}

	// Parse window
	if winMatch := regexp.MustCompile(`window\s*:\s*"([^"]+)"`).FindStringSubmatch(content); len(winMatch) > 1 {
		rl.Window = winMatch[1]
	}

	if rl.Requests > 0 || rl.Window != "" {
		return rl
	}

	return nil
}

func stringToInt(s string, result *int) (int, error) {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	*result = n
	return n, nil
}

// GetAllPermissions returns all parsed permissions.
func (p *Parser) GetAllPermissions() []*ServicePermissions {
	result := make([]*ServicePermissions, 0, len(p.services))
	for _, sp := range p.services {
		result = append(result, sp)
	}
	return result
}
