# Feature: jira issue worklog delete

## 📝 Resumen

Se ha implementado exitosamente la funcionalidad `jira issue worklog delete` que permite eliminar worklogs (registros de tiempo) de un issue de Jira.

## 🎯 Funcionalidad Implementada

### Comando CLI
```bash
jira issue worklog delete ISSUE-KEY [WORKLOG-ID] [flags]
```

### Flags disponibles
- `-f, --force` - Omite el prompt de confirmación
- `--help` - Muestra la ayuda del comando

### Alias
- `remove` - Alias alternativo para el comando `delete`
- `rm` - Alias corto para el comando `delete`

## 📂 Archivos Modificados/Creados

### 1. **pkg/jira/issue.go**
**Agregado:**
- Método `DeleteIssueWorklog()` - Elimina un worklog usando la API v2 de Jira

```go
// DELETE /rest/api/2/issue/{key}/worklog/{worklogID}
func (c *Client) DeleteIssueWorklog(key, worklogID string) error
```

### 2. **internal/cmd/issue/worklog/delete/delete.go** (NUEVO)
Implementa el comando CLI que:
- Parsea argumentos y flags
- Permite seleccionar el worklog interactivamente si no se especifica ID
- Solicita confirmación antes de eliminar (a menos que se use --force)
- Llama al cliente Jira para eliminar el worklog
- Muestra confirmación de éxito

### 3. **internal/cmd/issue/worklog/worklog.go**
**Modificado:**
- Agregado import del nuevo comando `delete`
- Registrado el comando `delete` en el comando padre `worklog`

## ✨ Funcionalidades

### 1. Eliminación directa con ID
```bash
jira issue worklog delete ISSUE-123 10001
```

### 2. Eliminación forzada (sin confirmación)
```bash
jira issue worklog delete ISSUE-123 10001 --force
```

### 3. Modo interactivo (selección de worklog)
```bash
jira issue worklog delete ISSUE-123
# El CLI te mostrará una lista de worklogs para seleccionar
```

### 4. Con proyecto por defecto
```bash
jira issue worklog delete 123 10001
```

## 🔧 Implementación Técnica

### API Endpoint Utilizado
```
DELETE /rest/api/2/issue/{issueIdOrKey}/worklog/{id}
```

### Compatibilidad
- **Jira Cloud**: ✅ Compatible
- **Jira Server**: ✅ Compatible (API v2)

### Flujo de Ejecución

1. **Parseo de argumentos**:
   - ISSUE-KEY (requerido)
   - WORKLOG-ID (opcional, se puede seleccionar interactivamente)

2. **Selección de worklog** (si no se proporcionó ID):
   - Obtiene todos los worklogs del issue
   - Muestra lista para selección
   - Usuario selecciona el worklog a eliminar

3. **Confirmación**:
   - Solicita confirmación al usuario
   - Se puede omitir con flag `--force`

4. **Eliminación**:
   - Envía DELETE a la API de Jira
   - Muestra mensaje de éxito

### Códigos de respuesta HTTP
- `204 No Content` - Eliminación exitosa
- `401 Unauthorized` - Sin permisos
- `404 Not Found` - Worklog o issue no encontrado

## 📊 Ejemplos de Uso

### Caso 1: Eliminación básica
```bash
$ jira issue worklog delete ISSUE-123 10001

? Are you sure you want to delete worklog 10001 from issue ISSUE-123? Yes
⠿ Deleting worklog
✓ Worklog deleted from issue "ISSUE-123"
https://your-domain.atlassian.net/browse/ISSUE-123
```

### Caso 2: Eliminación forzada (sin confirmación)
```bash
$ jira issue worklog delete ISSUE-123 10001 --force
✓ Worklog deleted from issue "ISSUE-123"
```

### Caso 3: Modo interactivo
```bash
$ jira issue worklog delete ISSUE-123

? Select worklog to delete:
  > 10001 - 2h 30m by John Doe (2024-11-05T10:30:00.000+0000)
    10002 - 1h 15m by Jane Smith (2024-11-05T14:00:00.000+0000)

? Are you sure you want to delete worklog 10001 from issue ISSUE-123? Yes
✓ Worklog deleted from issue "ISSUE-123"
```

### Caso 4: Usando alias remove
```bash
$ jira issue worklog remove ISSUE-123 10001
```

### Caso 5: Usando alias rm con force
```bash
$ jira issue worklog rm ISSUE-123 10001 -f
```

### Caso 6: Cancelar eliminación
```bash
$ jira issue worklog delete ISSUE-123 10001

? Are you sure you want to delete worklog 10001 from issue ISSUE-123? No
✗ Action cancelled
```

## 🔄 Integración con otros comandos

### Flujo completo: List → Delete
```bash
# 1. Listar worklogs para encontrar el ID
$ jira issue worklog list ISSUE-123

ID      AUTHOR          STARTED              TIME SPENT  CREATED
10001   John Doe        2024-11-05 10:30    2h 30m      2024-11-05 10:30
10002   Jane Smith      2024-11-05 14:00    1h 15m      2024-11-05 14:00

# 2. Eliminar el worklog no deseado
$ jira issue worklog delete ISSUE-123 10001
```

## ✅ Validaciones

El comando valida:
- ✅ Issue key válido
- ✅ Worklog ID existe
- ✅ Confirmación del usuario (a menos que se use --force)

## ⚠️ Consideraciones Importantes

1. **Acción irreversible**: Los worklogs eliminados NO se pueden recuperar
2. **Permisos**: El usuario debe tener permisos para eliminar worklogs en el issue
3. **Worklog de otros usuarios**: Dependiendo de la configuración de Jira, puede que no puedas eliminar worklogs de otros usuarios
4. **Estimación de tiempo**: Al eliminar un worklog, el tiempo estimado restante no se ajusta automáticamente

## 🛡️ Seguridad

### Confirmación obligatoria
Por defecto, el comando solicita confirmación antes de eliminar:
```
? Are you sure you want to delete worklog 10001 from issue ISSUE-123?
  > Yes
    No
```

### Modo force
Solo usa `--force` cuando estés completamente seguro:
```bash
# ⚠️ Elimina sin confirmación - usar con cuidado
jira issue worklog delete ISSUE-123 10001 --force
```

## 🔍 Troubleshooting

### Error: "Worklog not found"
**Causa**: El ID del worklog no existe o fue eliminado previamente

**Solución**: Usa `jira issue worklog list ISSUE-KEY` para obtener IDs válidos

### Error: "Permission denied"
**Causa**: No tienes permisos para eliminar el worklog

**Solución**: 
- Verifica que tienes permisos de edición en el issue
- Contacta al administrador de Jira si necesitas permisos adicionales
- Verifica si estás intentando eliminar un worklog de otro usuario

### Error: "Issue not found"
**Causa**: El issue key no existe o no tienes permisos para verlo

**Solución**: Verifica que el issue key sea correcto

## 📚 Comandos Relacionados

```bash
# Listar worklogs
jira issue worklog list ISSUE-KEY

# Agregar worklog
jira issue worklog add ISSUE-KEY "2h 30m"

# Editar worklog
jira issue worklog edit ISSUE-KEY WORKLOG-ID "3h"

# Eliminar worklog
jira issue worklog delete ISSUE-KEY WORKLOG-ID

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
- ✅ Confirmación de seguridad implementada

## 🚀 Próximas Mejoras Potenciales

1. **Bulk delete**: Eliminar múltiples worklogs a la vez
2. **Delete by criteria**: Eliminar worklogs por fecha, autor, etc.
3. **Soft delete**: Marcar como eliminado pero mantener en histórico
4. **Restore**: Deshacer eliminación reciente
5. **Audit log**: Registro de eliminaciones realizadas

## 📝 Notas de Implementación

### Compatibilidad
- Utiliza API v2 de Jira (compatible con Cloud y Server)
- Respeta el código de respuesta HTTP 204 (No Content)
- Maneja errores de permisos apropiadamente

### Seguridad
- Confirmación por defecto para prevenir eliminaciones accidentales
- Flag --force para scripts automatizados
- Mensajes claros sobre la acción a realizar

### Consistencia
- Sigue los mismos patrones que otros comandos worklog
- Modo interactivo coherente con `edit`
- Manejo de errores uniforme

## 💡 Casos de Uso

### 1. Corrección de errores
Eliminar worklogs registrados por error:
```bash
jira issue worklog delete ISSUE-123 10001
```

### 2. Limpieza de registros duplicados
```bash
# Listar para encontrar duplicados
jira issue worklog list ISSUE-123

# Eliminar el duplicado
jira issue worklog delete ISSUE-123 10002 --force
```

### 3. Scripts automatizados
```bash
#!/bin/bash
# Script para limpiar worklogs antiguos
ISSUE_KEY="ISSUE-123"
WORKLOG_IDS=("10001" "10002" "10003")

for id in "${WORKLOG_IDS[@]}"; do
    jira issue worklog delete "$ISSUE_KEY" "$id" --force
done
```

## 🔗 Referencias

- [Jira REST API - Delete worklog](https://developer.atlassian.com/cloud/jira/platform/rest/v2/api-group-issue-worklogs/#api-rest-api-2-issue-issueidorkey-worklog-id-delete)
- [Jira Time Tracking](https://support.atlassian.com/jira-cloud-administration/docs/configure-time-tracking/)
- [Jira Permissions](https://support.atlassian.com/jira-cloud-administration/docs/manage-project-permissions/)

---

**Fecha de implementación:** 2024-11-05  
**Versión:** Compatible con jira-cli v1.x  
**Estado:** ✅ Completado y funcional  
**Dependencias:** Requiere `worklog list` para modo interactivo
