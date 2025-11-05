# 📋 Análisis: Cómo se hacen las peticiones a Jira Cloud en jira-cli

## 🏗️ Arquitectura General

El proyecto está organizado en 3 capas principales:

1. **`api/` (Capa de Abstracción)** - Inicializa el cliente y gestiona las diferencias entre Cloud y On-Premise
2. **`pkg/jira/` (Capa Core)** - Implementa toda la lógica HTTP y comunicación con la API de Jira
3. **`cmd/` (Capa CLI)** - Comandos de usuario que invocan las capas anteriores

---

## 🔑 Flujo de Autenticación

El cliente se configura con múltiples fuentes de credenciales (en orden de prioridad):

```go
// Ubicación: api/client.go
func Client(config jira.Config) *jira.Client {
    // 1. Desde configuración explícita
    config.Server = viper.GetString("server")
    config.Login = viper.GetString("login")
    config.APIToken = viper.GetString("api_token")
    
    // 2. Desde archivo .netrc
    if config.APIToken == "" {
        netrcConfig, _ := netrc.Read(config.Server, config.Login)
        if netrcConfig != nil {
            config.APIToken = netrcConfig.Password
        }
    }
    
    // 3. Desde keychain del sistema
    if config.APIToken == "" {
        secret, _ := keyring.Get("jira-cli", config.Login)
        config.APIToken = secret
    }
    
    // Soporta 3 tipos de autenticación:
    // - Basic Auth (usuario + API token)
    // - Bearer (Personal Access Token)
    // - mTLS (certificados de cliente)
}
```

### Variables de Entorno

- `JIRA_API_TOKEN` - Token de API o contraseña
- `JIRA_AUTH_TYPE` - Tipo de autenticación (basic, bearer, mtls)
- `JIRA_CONFIG_FILE` - Ubicación del archivo de configuración

---

## 🌐 Cliente HTTP (pkg/jira/client.go)

El **corazón del sistema** es la estructura `Client`:

```go
type Client struct {
    transport http.RoundTripper
    server    string          // URL base de Jira
    login     string           // Usuario
    token     string           // API Token o PAT
    authType  *AuthType        // basic, bearer o mtls
    timeout   time.Duration
    debug     bool
}
```

### Métodos de peticiones por versión de API

- **V3 (Jira Cloud)**: `Get()`, `Post()`, `Put()`
- **V2 (Jira Server/Local)**: `GetV2()`, `PostV2()`, `PutV2()`
- **V1 (Agile API)**: `GetV1()`, `PostV1()`, `PutV1()`

### Rutas base

```go
baseURLv3 = "/rest/api/3"       // Cloud moderno
baseURLv2 = "/rest/api/2"       // Server/Cloud legacy
baseURLv1 = "/rest/agile/1.0"   // Agile endpoints
```

### Configuración del Transport HTTP

```go
transport := &http.Transport{
    Proxy: http.ProxyFromEnvironment,
    TLSClientConfig: &tls.Config{
        MinVersion:         tls.VersionTLS12,
        InsecureSkipVerify: client.insecure,
    },
    DialContext: (&net.Dialer{
        Timeout: client.timeout,
    }).DialContext,
}
```

---

## 🔄 Ejemplo Completo: Búsqueda de Issues

### 1. Desde el CLI → API → Cliente HTTP

```go
// Desde un comando CLI se invoca:
api.ProxySearch(client, jql, from, limit)
    ↓
// api/client.go decide qué versión usar:
func ProxySearch(c *jira.Client, jql string, from, limit uint) (*jira.SearchResult, error) {
    it := viper.GetString("installation")
    
    if it == jira.InstallationTypeLocal {
        return c.SearchV2(jql, from, limit)  // v2 para Server
    }
    return c.Search(jql, limit)              // v3 para Cloud
}
    ↓
// pkg/jira/search.go construye la petición:
func (c *Client) Search(jql string, limit uint) (*SearchResult, error) {
    path := fmt.Sprintf("/search/jql?jql=%s&maxResults=%d&fields=*all", 
                        url.QueryEscape(jql), limit)
    return c.search(path, apiVersion3)
}
    ↓
// pkg/jira/client.go ejecuta GET HTTP:
func (c *Client) Get(ctx context.Context, path string, headers Header) (*http.Response, error) {
    return c.request(ctx, http.MethodGet, c.server+baseURLv3+path, nil, headers)
}
    ↓
// Método request() añade autenticación:
func (c *Client) request(ctx context.Context, method, endpoint string, body []byte, headers Header) (*http.Response, error) {
    req, _ := http.NewRequest(method, endpoint, bytes.NewReader(body))
    
    switch c.authType.String() {
        case string(AuthTypeBasic):
            req.SetBasicAuth(c.login, c.token)
        case string(AuthTypeBearer):
            req.Header.Add("Authorization", "Bearer " + c.token)
        case string(AuthTypeMTLS):
            if c.token != "" {
                req.Header.Add("Authorization", "Bearer " + c.token)
            }
    }
    
    httpClient := &http.Client{Transport: c.transport}
    return httpClient.Do(req.WithContext(ctx))
}
```

---

## 📝 Ejemplo: Crear un Issue

```go
// pkg/jira/create.go
func (c *Client) Create(req *CreateRequest) (*CreateResponse, error) {
    // 1. Construye el payload JSON
    data := c.getRequestData(req)
    body, _ := json.Marshal(data)
    
    // 2. POST a /rest/api/3/issue
    res, err := c.Post(context.Background(), "/issue", body, Header{
        "Content-Type": "application/json",
    })
    
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()
    
    // 3. Valida respuesta
    if res.StatusCode != http.StatusCreated {
        return nil, formatUnexpectedResponse(res)
    }
    
    // 4. Parsea respuesta
    var out CreateResponse
    json.NewDecoder(res.Body).Decode(&out)
    return &out, err
}
```

### Estructura de CreateRequest

```go
type CreateRequest struct {
    Project          string
    Name             string
    IssueType        string
    ParentIssueKey   string
    Summary          string
    Body             interface{} // string en v2, adf.ADF en v3
    Reporter         string
    Assignee         string
    Priority         string
    Labels           []string
    Components       []string
    FixVersions      []string
    AffectsVersions  []string
    OriginalEstimate string
    CustomFields     map[string]string
}
```

---

## 📚 Ejemplo: Obtener un Issue

```go
// pkg/jira/issue.go
func (c *Client) GetIssue(key string, opts ...filter.Filter) (*Issue, error) {
    path := fmt.Sprintf("/issue/%s", key)
    
    res, err := c.Get(context.Background(), path, nil)
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()
    
    if res.StatusCode != http.StatusOK {
        return nil, formatUnexpectedResponse(res)
    }
    
    var iss Issue
    err = json.NewDecoder(res.Body).Decode(&iss)
    
    // Convierte descripción de ADF a formato legible
    iss.Fields.Description = ifaceToADF(iss.Fields.Description)
    
    return &iss, err
}
```

---

## 🛡️ Características de Seguridad

### 1. TLS 1.2+ obligatorio

```go
TLSClientConfig: &tls.Config{
    MinVersion:         tls.VersionTLS12,
    InsecureSkipVerify: client.insecure,
}
```

### 2. Soporte mTLS con certificados de cliente

```go
if c.AuthType == AuthTypeMTLS {
    caCert, _ := os.ReadFile(c.MTLSConfig.CaCert)
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    cert, _ := tls.LoadX509KeyPair(c.MTLSConfig.ClientCert, c.MTLSConfig.ClientKey)
    
    transport.TLSClientConfig.RootCAs = caCertPool
    transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
}
```

### 3. Múltiples fuentes de credenciales

- Variables de entorno
- Archivo de configuración (viper)
- Archivo `.netrc`
- Keychain del sistema operativo

### 4. Soporte de Proxy

```go
Proxy: http.ProxyFromEnvironment
```

---

## 🔍 Manejo de Errores Personalizados

```go
type ErrUnexpectedResponse struct {
    Body       Errors
    Status     string
    StatusCode int
}

type Errors struct {
    Errors          map[string]string
    ErrorMessages   []string
    WarningMessages []string
}

func (e Errors) String() string {
    var out strings.Builder
    
    if len(e.ErrorMessages) > 0 || len(e.Errors) > 0 {
        out.WriteString("\nError:\n")
        for _, v := range e.ErrorMessages {
            out.WriteString(fmt.Sprintf("  - %s\n", v))
        }
        for k, v := range e.Errors {
            out.WriteString(fmt.Sprintf("  - %s: %s\n", k, v))
        }
    }
    
    return out.String()
}
```

---

## 🔄 Patrón Proxy para Compatibilidad

El archivo `api/client.go` implementa funciones **Proxy** que determinan automáticamente qué versión de la API usar:

```go
func ProxyGetIssue(c *jira.Client, key string, opts ...filter.Filter) (*jira.Issue, error) {
    it := viper.GetString("installation")
    
    if it == jira.InstallationTypeLocal {
        return c.GetIssueV2(key, opts...)  // Jira Server
    }
    return c.GetIssue(key, opts...)         // Jira Cloud
}

func ProxyAssignIssue(c *jira.Client, key string, user *jira.User, def string) error {
    it := viper.GetString("installation")
    assignee := def
    
    if user != nil {
        switch it {
        case jira.InstallationTypeLocal:
            assignee = user.Name      // Server usa 'name'
        default:
            assignee = user.AccountID // Cloud usa 'accountId'
        }
    }
    
    if it == jira.InstallationTypeLocal {
        return c.AssignIssueV2(key, assignee)
    }
    return c.AssignIssue(key, assignee)
}
```

---

## 📌 Resumen del Flujo de Peticiones

```
┌─────────────────┐
│  CLI Command    │
│  (cmd/)         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  api.Proxy*()   │ ◄── Decide v2 vs v3 según instalación
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  pkg/jira/*.go  │ ◄── Construye endpoint + payload
│  - search.go    │
│  - issue.go     │
│  - create.go    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  client.go      │ ◄── Añade headers de autenticación
│  Get/Post/Put() │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  http.Client    │ ◄── Ejecuta HTTP request
│  .Do()          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  JSON Decode    │ ◄── Parsea respuesta a structs Go
│  → Struct Go    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Respuesta CLI  │
└─────────────────┘
```

---

## 🎯 Ventajas del Diseño

✅ **Abstracción clara** entre Cloud/Server  
✅ **Reutilización de código HTTP** con métodos genéricos  
✅ **Testing fácil** (mocking del cliente)  
✅ **Soporte múltiples métodos** de autenticación  
✅ **Gestión centralizada** de timeouts y configuración TLS  
✅ **Compatibilidad automática** con diferentes versiones de Jira  
✅ **Manejo robusto de errores** con tipos personalizados  

---

## 📦 Dependencias Principales

```go
// HTTP y Red
net/http
crypto/tls
context

// JSON
encoding/json

// Configuración
github.com/spf13/viper
github.com/mitchellh/go-homedir

// Seguridad
github.com/zalando/go-keyring
pkg/netrc (interno)

// Utilidades
github.com/fatih/color
github.com/briandowns/spinner
```

---

## 🔧 Configuración del Cliente

### Timeout

```go
const clientTimeout = 15 * time.Second

jira.NewClient(
    config,
    jira.WithTimeout(clientTimeout),
)
```

### Debug Mode

Cuando `debug: true` está habilitado, se imprimen:
- Request completo (headers, body)
- Response completo (headers, status)

```go
if c.debug {
    dump(req, res)
}
```

---

## 📄 Endpoints Principales Implementados

### Issues
- `GET /rest/api/3/issue/{key}` - Obtener issue
- `POST /rest/api/3/issue` - Crear issue
- `PUT /rest/api/3/issue/{key}` - Editar issue
- `DELETE /rest/api/2/issue/{key}` - Eliminar issue
- `PUT /rest/api/3/issue/{key}/assignee` - Asignar issue
- `GET /rest/api/3/issue/{key}/transitions` - Obtener transiciones
- `POST /rest/api/3/issue/{key}/transitions` - Transicionar issue

### Worklogs
- `GET /rest/api/2/issue/{key}/worklog` - Listar worklogs de un issue
- `POST /rest/api/2/issue/{key}/worklog` - Añadir worklog a un issue
- `PUT /rest/api/2/issue/{key}/worklog/{id}` - Actualizar worklog de un issue
- `DELETE /rest/api/2/issue/{key}/worklog/{id}` - Eliminar worklog de un issue

### Búsqueda
- `GET /rest/api/3/search/jql` - Búsqueda con JQL (v3)
- `GET /rest/api/2/search` - Búsqueda con JQL (v2)

### Usuarios
- `GET /rest/api/3/user/assignable/search` - Buscar usuarios asignables

### Epics y Sprints
- `GET /rest/agile/1.0/epic/{id}/issue` - Issues de un epic
- `GET /rest/agile/1.0/sprint/{id}/issue` - Issues de un sprint
- `POST /rest/agile/1.0/sprint/{id}/issue` - Añadir issues al sprint

### Proyectos
- `GET /rest/api/3/project` - Listar proyectos
- `GET /rest/api/3/project/{key}` - Obtener proyecto

### Boards
- `GET /rest/agile/1.0/board` - Listar boards

---

## 🧪 Testing

El proyecto incluye tests unitarios para cada módulo:

```
pkg/jira/
├── client_test.go
├── issue_test.go
├── search_test.go
├── create_test.go
└── testdata/
```

Uso de mocks para simular respuestas HTTP sin hacer peticiones reales.

---

**Análisis generado el:** 2025-11-05  
**Repositorio:** git@github.com:gvcrescitelli/jira-cli.git  
**Lenguaje:** Go 1.24+
