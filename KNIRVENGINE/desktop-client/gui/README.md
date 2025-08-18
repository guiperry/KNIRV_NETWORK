# Inference Engine GUI

This is the frontend application for the Inference Engine project.

## Development

To start the development server:

```bash
npm run dev
```

## Building for Production

To build the application for production:

```bash
npm run build
```

This will create a `dist` directory with the compiled static files. These files are automatically served by the Go backend when running in production mode.

## Production Deployment

To run the entire application in production mode:

1. Build the frontend:
   ```bash
   cd gui
   npm run build
   cd ..
   ```

2. Run the Go server with the production flag:
   ```bash
   go run main.go --production
   ```

Alternatively, use the provided script:
```bash
./build_and_run.sh
```

## Configuration

The build configuration is defined in `vite.config.ts`. The application is configured to output static files to the `dist` directory, which is then served by the Go backend.