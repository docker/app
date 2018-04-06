package utils

import (
    "strings"
)

var keywords = map[string]string{

    // databases
    "mysql":      "mysql",
    "msql":       "mysql",
    "postgres":   "postgres",
    "postgresql": "postgres",
    "mongo":      "mongo",
    "mongodb":    "mongo",
    "maria":      "mariadb",
    "mariadb":    "mariadb",

    // proxies
    "traefik": "traefik",
    "trafik":  "traefik",
    "nginx":   "nginx",
    "apache":  "httpd",
    "httpd":   "httpd",
    "haproxy": "haproxy",

    // caches
    "redis":     "redis",
    "memcache":  "memcached",
    "memcached": "memcached",

    // applicative
    "tomcat":        "tomcat",
    "jetty":         "jetty",
    "java":          "java",
    "jdk":           "openjdk",
    "openjdk":       "openjdk",
    "python":        "python",
    "rails":         "rails",
    "ruby on rails": "rails",
    "ruby":          "ruby",
    "mono":          "mono",
    "django":        "django",
    "jango":         "django",
    "php":           "php",
    "symfony":       "php",
    "go":            "golang",
    "golang":        "golang",
    "wordpress":     "wordpress",
    "node":          "node",
    "nodejs":        "node",

    // if all else fails
    "default": "alpine",
}

// MatchService matches user input with a viable service config stub
func MatchService(inputName string) ServiceConfig {
    inputName = strings.ToLower(inputName)
    imgName, ok := keywords[inputName]
    if !ok {
        imgName = keywords["default"]
    }
    return ServiceConfig{
        ServiceName:  strings.Replace(inputName, " ", "_", -1),
        ServiceImage: imgName,
    }
}

// ServiceConfig is a stub containing basic information about a service component
type ServiceConfig struct {
    ServiceName  string
    ServiceImage string
}
