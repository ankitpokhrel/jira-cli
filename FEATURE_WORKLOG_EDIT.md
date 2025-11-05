# Feature: jira issue worklog edit

## 📝 Resumen

Se ha implementado exitosamente la funcionalidad `jira issue worklog edit` que permite editar worklogs (registros de tiempo) existentes en un issue de Jira.

## 🎯 Funcionalidad Implementada

### Comando CLI
```bash
jira issue worklog edit ISSUE-KEY WORKLOG-ID TIME_SPENT [flags]
```

### Flags disponibles
- `--started string` - Nueva fecha/hora de inicio del trabajo
- `--timezone string` - Zona horaria para la fecha (default: "UTC")
- `--comment string` - Nuevo comentario para el worklog
- `--no-input` - Deshabilitar prompts interactivos
- `--help` - Muestra la ayuda del comando

### Alias
- `update` - Alias alternativo para el comando `edit`

## 📂 Archivos Modificados/Creados

### 1. **pkg/jira/issue.go**
**Agregado:**
- Método `UpdateIssueWorklog()` - Actualiza un worklog usando la API v2 de Jira

```go
// PUT /rest/api/2/issue/{key}/worklog/{worklogID}
func (c *Client) UpdateIssueWorklog(key, worklogID, started, timeSpent, comment string) error
```

### 2. **internal/cmd/issue/worklog/edit/edit.go** (NUEVO)
Implementa el comando CLI que:
- Parsea argumentos y flags
- Permite seleccionar el worklog interactivamente si no se especifica ID
- Valida los parámetros
- Llama al cliente Jira para actualizar el worklog
- Muestra confirmación de éxito

### 3. **internal/cmd/issue/worklog/worklog.go**
**Modificado:**
- Agregado import del nuevo comando `edit`
- Registrado el comando `edit` en el comando padre `worklog`

## 🔍 Estructura de Datos

### UpdateIssueWorklog Request
```go
{
  "timeSpent": "3h 30m",      // Nuevo tiempo registrado
  "comment": "Updated work",   // Nuevo comentario (opcional)
  "started": "2024-11-05T10:30:00.000+0000"  // Nueva fecha inicio (opcional)
}
```

## ✨ Funcionalidades

### 1. Edición con todos los parámetros
```bash
jira issue worklog edit ISSUE-123 10001 "3h 30m" \
  --comment "Updated work description" \
  --started "2024-11-05 09:30:00"
```

### 2. Edición solo del tiempo
```bash
jira issue worklog edit ISSUE-123 10001 "4h" --no-input
```

### 3. Modo interactivo (selección de worklog)
```bash
jira issue worklog edit ISSUE-123
# El CLI te mostrará una lista de worklogs para seleccionar
```

### 4. Con zona horaria personalizada
```bash
jira issue worklog edit ISSUE-123 10001 "2h" \
  --started "2024-11-05 14:00:00" \
  --timezone "Europe/Madrid"
```

## 🔧 Implementación Técnica

### API Endpoint Utilizado
```
PUT /rest/api/2/issue/{issueIdOrKey}/worklog/{id}
```

### Compatibilidad
- **Jira Cloud**: ✅ Compatible
- **Jira Server**: ✅ Compatible (API v2)

### Flujo de Ejecución

1. **Parseo de argumentos**:
   - ISSUE-KEY (requerido)
   - WORKLOG-ID (opcional, se puede seleccionar interactivamente)
   - TIME_SPENT (requerido)

2. **Validación**:
   - Si no se proporciona WORKLOG-ID, se obtienen los worklogs del issue
   - El usuario selecciona el worklog a editar de una lista

3. **Prompts interactivos** (si `--no-input` no está presente):
   - Solicita TIME_SPENT si no se proporcionó
   - Solicita comentario opcional
   - Confirma la acción

4. **Actualización**:
   - Construye el request con los nuevos valores
   - Envía PUT a la API de Jira
   - Muestra mensaje de éxito

## 📊 Ejemplos de Uso

### Caso 1: Editar tiempo y comentario
```bash
$ jira issue worklog edit ISSUE-123 10001 "5h" --comment "Completed feature implementation"
✓ Worklog updated in issue "ISSUE-123"
https://your-domain.atlassian.net/browse/ISSUE-123
```

### Caso 2: Cambiar fecha de inicio
```bash
$ jira issue worklog edit ISSUE-123 10001 "3h" --started "2024-11-04 10:00:00"
✓ Worklog updated in issue "ISSUE-123"
```

### Caso 3: Modo interactivo
```bash
$ jira issue worklog edit ISSUE-123

? Select worklog to edit:
  > 10001 - 2h 30m by John Doe (2024-11-05T10:30:00.000+0000)
    10002 - 1h 15m by Jane Smith (2024-11-05T14:00:00.000+0000)

? Time spent: 4h
? Comment body: [Editor opens]
? What's next? 
  > Submit
    Cancel

✓ Worklog updated in issue "ISSUE-123"
```

### Caso 4: Usando alias
```bash
$ jira issue worklog update ISSUE-123 10001 "6h"
```

### Caso 5: Con proyecto por defecto
```bash
# Si tienes un proyecto configurado
$ jira issue worklog edit 123 10001 "2h"
```

## 🔄 Integración con otros comandos

### Flujo completo: List → Edit
```bash
# 1. Listar worklogs para encontrar el ID
$ jira issue worklog list ISSUE-123

ID      AUTHOR          STARTED              TIME SPENT  CREATED
10001   John Doe        2024-11-05 10:30    2h 30m      2024-11-05 10:30
10002   Jane Smith      2024-11-05 14:00    1h 15m      2024-11-05 14:00

# 2. Editar el worklog deseado
$ jira issue worklog edit ISSUE-123 10001 "3h"
```

## ✅ Validaciones

El comando valida:
- ✅ Issue key válido
- ✅ Worklog ID existe
- ✅ Formato de tiempo válido (Xd Xh Xm)
- ✅ Formato de fecha válido (si se proporciona)
- ✅ Zona horaria válida en formato IANA

## ⚠️ Consideraciones

1. **Permisos**: El usuario debe tener permisos para editar worklogs en el issue
2. **Worklog de otros usuarios**: Dependiendo de la configuración de Jira, puede que no puedas editar worklogs de otros usuarios
3. **Formato de tiempo**: Debe seguir el formato de Jira (Xd Xh Xm)
4. **Fechas**: Se pueden especificar con o sin zona horaria

## 🔍 Troubleshooting

### Error: "Worklog not found"
**Causa**: El ID del worklog no existe o fue eliminado

**Solución**: Usa `jira issue worklog list ISSUE-KEY` para obtener IDs válidos

### Error: "Permission denied"
**Causa**: No tienes permisos para editar el worklog

**Solución**: Verifica tus permisos en Jira o contacta al administrador

### Error: "Invalid time format"
**Causa**: El formato de tiempo no es válido

**Solución**: Usa formato Jira: "2h 30m", "1d 4h", etc.

## 📚 Comandos Relacionados

```bash
# Listar worklogs
jira issue worklog list ISSUE-KEY

# Agregar worklog
jira issue worklog add ISSUE-KEY "2h 30m"

# Editar worklog
jira issue worklog edit ISSUE-KEY WORKLOG-ID "3h"

# Ver issue completo
jira issue view ISSUE-KEY
```

## 🧪 Testing

- ✅ Compilación exitosa sin errores
- ✅ Todos los tests existentes pasan
- ✅ Comando registrado correctamente
- ✅ Help text funciona correctamente
- ✅ Flags parseados correctamente
- ✅ Integrado con modo interactivo

## 🚀 Próximas Mejoras Potenciales

1. **Delete worklog**: Comando para eliminar worklogs
2. **Bulk edit**: Editar múltiples worklogs a la vez
3. **Copy worklog**: Copiar un worklog a otro issue
4. **Export worklogs**: Exportar a CSV/JSON para análisis
5. **Templates**: Plantillas para comentarios frecuentes

## 📝 Notas de Implementación

### Compatibilidad
- Utiliza API v2 de Jira (compatible con Cloud y Server)
- Maneja formatos de fecha con y sin timezone
- Soporta markdown para comentarios

### Consistencia
- Sigue los mismos patrones que `worklog add`
- Reutiliza estructuras existentes (`issueWorklogRequest`)
- Mismo flujo interactivo que otros comandos

### Performance
- Una sola petición HTTP para actualizar
- Cacheo de lista de worklogs para selección interactiva
- Validación temprana de parámetros

## 🔗 Referencias

- [Jira REST API - Update worklog](https://developer.atlassian.com/cloud/jira/platform/rest/v2/api-group-issue-worklogs/#api-rest-api-2-issue-issueidorkey-worklog-id-put)
- [Jira Time Tracking](https://support.atlassian.com/jira-cloud-administration/docs/configure-time-tracking/)

---

**Fecha de implementación:** 2024-11-05  
**Versión:** Compatible con jira-cli v1.x  
**Estado:** ✅ Completado y funcional  
**Dependencias:** Requiere `worklog list` para modo interactivo
