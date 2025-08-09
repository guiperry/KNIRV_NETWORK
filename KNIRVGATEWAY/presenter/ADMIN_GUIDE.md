# KNIRV Presenter Admin Guide

## 🔐 Import Password Management

The KNIRV Presenter system includes password protection for the import functionality to prevent unauthorized access to presentation management features.

### Password Configuration

The import password is stored in `import-config.json`:

```json
{
  "importPassword": "knirv2024import",
  "description": "Password required to access the presentation import functionality",
  "lastUpdated": "2024-08-09"
}
```

### Changing the Import Password

1. **Edit the configuration file**:
   ```bash
   nano import-config.json
   ```

2. **Update the password**:
   ```json
   {
     "importPassword": "your_new_secure_password",
     "description": "Password required to access the presentation import functionality",
     "lastUpdated": "2024-08-09"
   }
   ```

3. **Save and deploy**:
   - Commit the changes to your repository
   - Netlify will automatically deploy the updated configuration

### Security Best Practices

- **Use strong passwords**: Combine letters, numbers, and special characters
- **Regular updates**: Change the password periodically
- **Secure sharing**: Only share the password with authorized users
- **Monitor access**: Keep track of who has import access

### Access Control

#### How It Works:
1. **Button Protection**: Import button requires password authentication
2. **Page Protection**: Direct access to `import.html` is blocked without authorization
3. **Session Management**: Access is granted for the browser session only
4. **Automatic Redirect**: Unauthorized access redirects to the main page

#### User Experience:
1. User clicks "Import Presentations" button
2. Password modal appears
3. User enters import password
4. On success: Redirected to import page
5. On failure: Error message displayed

### Troubleshooting

#### Common Issues:

**"Import configuration not found"**
- Ensure `import-config.json` exists in the presenter directory
- Check file permissions and accessibility

**"Incorrect import password"**
- Verify the password in `import-config.json`
- Check for typos or extra spaces
- Passwords are case-sensitive

**Direct access blocked**
- This is expected behavior for security
- Users must use the Import button on the main page

### File Structure

```
presenter/
├── import-config.json     # Import password configuration
├── index.html            # Main page with protected import button
├── import.html           # Protected import page
├── presentations/        # Presentation storage
└── ...
```

### Deployment Notes

- **Static hosting**: Configuration works with Netlify static hosting
- **No server required**: All authentication is client-side
- **Instant updates**: Changes deploy automatically with Netlify
- **Cross-platform**: Works on all devices and browsers

### Backup and Recovery

1. **Backup configuration**:
   ```bash
   cp import-config.json import-config.backup.json
   ```

2. **Version control**: Keep configuration in your repository

3. **Recovery**: Restore from backup if needed:
   ```bash
   cp import-config.backup.json import-config.json
   ```

### Integration with KNIRVGATEWAY

The presenter system is fully integrated with the KNIRVGATEWAY deployment:

- **Automatic deployment**: Changes deploy with the main site
- **Consistent security**: Follows KNIRV security patterns
- **Centralized management**: All configuration in one repository

For additional security questions or advanced configuration, refer to the main KNIRVGATEWAY documentation.
