# Feature: jira issue worklog list

## 📝 Resumen

Se ha implementado exitosamente la funcionalidad `jira issue worklog list` que permite listar todos los worklogs (registros de tiempo) de un issue específico en Jira.

## 🎯 Funcionalidad Implementada

### Comando CLI
```bash
jira issue worklog list ISSUE-KEY [flags]
```

### Flags disponibles
- `--plain`: Muestra la salida en modo texto plano con detalles completos
- `--table`: Muestra la salida en formato tabla compacta
- `--help`: Muestra la ayuda del comando

### Alias
- `ls` - Alias corto para el comando `list`

## 📂 Archivos Modificados/Creados

### 1. **pkg/jira/issue.go**
**Agregado:**
- Tipo `Worklog` - Estructura que representa un worklog
- Tipo `WorklogResponse` - Estructura para la respuesta de la API
- Método `GetIssueWorklogs()` - Obtiene los worklogs de un issue usando la API v2 de Jira

```go
// GET /rest/api/2/issue/{key}/worklog
func (c *Client) GetIssueWorklogs(key string) (*WorklogResponse, error)
```

### 2. **internal/cmd/issue/worklog/list/list.go** (NUEVO)
Implementa el comando CLI que:
- Parsea argumentos y flags
- Llama al cliente Jira para obtener los worklogs
- Renderiza la salida usando el sistema de vistas

### 3. **internal/cmd/issue/worklog/worklog.go**
**Modificado:**
- Agregado import del nuevo comando `list`
- Registrado el comando `list` en el comando padre `worklog`

### 4. **internal/view/worklog.go** (NUEVO)
Implementa la vista para mostrar worklogs con dos formatos:
- **Plain mode**: Detalles completos de cada worklog
- **Table mode**: Vista compacta en formato tabla

## 🔍 Estructura de Datos

### Worklog
```go
type Worklog struct {
    ID               string      // ID único del worklog
    IssueID          string      // ID del issue
    Author           User        // Usuario que creó el worklog
    UpdateAuthor     User        // Usuario que actualizó el worklog
    Comment          interface{} // Comentario (string o ADF)
    Created          string      // Fecha de creación
    Updated          string      // Fecha de actualización
    Started          string      // Fecha de inicio del trabajo
    TimeSpent        string      // Tiempo gastado (formato legible: 2h 30m)
    TimeSpentSeconds int         // Tiempo gastado en segundos
}
```

## 📊 Formatos de Salida

### Modo Plain (--plain)
```
Worklog #1
  ID:          12345
  Author:      John Doe
  Started:     2024-11-05 10:30
  Time Spent:  2h 30m (9000 seconds)
  Created:     2024-11-05 10:30
  Updated:     2024-11-05 10:30
  Comment:     Working on feature implementation

Worklog #2
  ID:          12346
  Author:      Jane Smith
  Started:     2024-11-05 14:00
  Time Spent:  1h 15m (4500 seconds)
  Created:     2024-11-05 14:00
  Updated:     2024-11-05 14:00

Total worklogs: 2
```

### Modo Table (default)
```
ID      AUTHOR          STARTED              TIME SPENT  CREATED
12345   John Doe        2024-11-05 10:30    2h 30m      2024-11-05 10:30
12346   Jane Smith      2024-11-05 14:00    1h 15m      2024-11-05 14:00
```

## 🔧 Características Técnicas

### Endpoint API Utilizado
```
GET /rest/api/2/issue/{issueKey}/worklog
```

### Manejo de Fechas
Las fechas son parseadas desde múltiples formatos Jira y convertidas a formato legible:
- RFC3339
- RFC3339 con milisegundos
- Formato personalizado de Jira

### Manejo de Comentarios
Soporta tanto comentarios en texto plano como en formato ADF (Atlassian Document Format):
- **Texto plano**: Se muestra directamente
- **ADF**: Se extrae el texto recursivamente del árbol de nodos

## 📖 Ejemplos de Uso

### Listar worklogs en formato tabla (default)
```bash
jira issue worklog list ISSUE-123
```

### Listar worklogs con detalles completos
```bash
jira issue worklog list ISSUE-123 --plain
```

### Usando alias
```bash
jira issue worklog ls ISSUE-123
```

### Con proyecto configurado
```bash
jira issue worklog list 123  # Usa el proyecto por defecto
```

## ✅ Testing

- ✅ Compilación exitosa sin errores
- ✅ Todos los tests existentes pasan
- ✅ Comando registrado correctamente en la CLI
- ✅ Help text funciona correctamente
- ✅ Flags parseados correctamente

## 🚀 Próximas Mejoras Potenciales

1. **Filtrado**: Agregar filtros por autor, rango de fechas
2. **Ordenamiento**: Permitir ordenar por fecha, autor, tiempo
3. **Paginación**: Para issues con muchos worklogs
4. **Export**: Exportar a CSV o JSON
5. **Totales**: Mostrar suma total de tiempo registrado
6. **Delete**: Comando para eliminar worklogs
7. **Edit**: Comando para editar worklogs existentes

## 📝 Notas de Implementación

### Compatibilidad
- Utiliza API v2 de Jira (compatible con Cloud y Server)
- Maneja tanto formato de comentarios legacy (string) como moderno (ADF)
- Respeta la configuración de proyecto y servidor del usuario

### Consistencia
- Sigue los patrones establecidos en el proyecto
- Utiliza las mismas estructuras de DisplayFormat que otros comandos
- Reutiliza utilidades existentes (cmdutil, view helpers)

### Performance
- Obtiene todos los worklogs en una sola petición
- No hay límite artificial en el número de worklogs mostrados
- Renderizado eficiente usando buffers

## 🔗 Referencias

- [Jira REST API - Get issue worklogs](https://developer.atlassian.com/cloud/jira/platform/rest/v2/api-group-issue-worklogs/#api-rest-api-2-issue-issueidorkey-worklog-get)
- [Atlassian Document Format (ADF)](https://developer.atlassian.com/cloud/jira/platform/apis/document/structure/)

---

**Fecha de implementación:** 2025-11-05  
**Versión:** Compatible con jira-cli v1.x  
**Estado:** ✅ Completado y funcional
