# Metrist Datasource for Grafana

## Getting started

Add the golang and mage plugin through asdf and install the required versions stated in `.tool-versions`

```bash
 asdf plugin-add golang
 asdf plugin-add mage
 asdf install
```

Quickest way to get started is to run 

```bash
npm install npm@8 -g # Make sure you have NPM version 8 to avoid grafana-e2e cli issue
npm install
go get
mage -v         # Build the backend
npm run dev     # Build the frontend
npm run server  # Run the grana instance
```


### Frontend

1. Install dependencies

   ```bash
   npm install
   ```

2. Build plugin in development mode or run in watch mode

   ```bash
   npm run dev

   # or

   npm run watch
   ```

3. Build plugin in production mode

   ```bash
   npm run build
   ```

4. Run the tests (using Jest)

   ```bash
   # Runs the tests and watches for changes
   npm run test
   
   # Exists after running all the tests
   npm run lint:ci
   ```

5. Spin up a Grafana instance and run the plugin inside it (using Docker)

   ```bash
   npm run server
   ```

6. Run the E2E tests (using Cypress)

   ```bash
   # Spin up a Grafana instance first that we tests against 
   npm run server
   
   # Start the tests
   npm run e2e
   ```

7. Run the linter

   ```bash
   npm run lint
   
   # or

   npm run lint:fix
   ```

### Backend

1. Update [Grafana plugin SDK for Go](https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/) dependency to the latest minor version:

   ```bash
   go get
   ```

2. Build backend plugin binaries for Linux, Windows and Darwin:

   ```bash
   mage -v
   ```

3. List all available Mage targets for additional commands:

   ```bash
   mage -l
   ```

## Learn more

Below you can find source code for existing app plugins and other related documentation.

- [Basic data source plugin example](https://github.com/grafana/grafana-plugin-examples/tree/master/examples/datasource-basic#readme)
- [Plugin.json documentation](https://grafana.com/docs/grafana/latest/developers/plugins/metadata/)
- [How to sign a plugin?](https://grafana.com/docs/grafana/latest/developers/plugins/sign-a-plugin/)
