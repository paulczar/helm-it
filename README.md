# Helm It
![Helm Template Service Logo](logo/helmit-readme.png)

This service provides a web-based API and user interface to render Helm charts without requiring the client to have Helm installed locally. It is a stateless service that, given a URL to a Helm chart and a set of values, returns the rendered Kubernetes manifests.

## Features

*   **Web Interface**: A simple web form to input a chart URL and values.
*   **Syntax Highlighting**: The rendered YAML output is displayed with syntax highlighting.
*   **Copy and Download**: Buttons to easily copy the output to the clipboard or download it as a file.
*   **Automatic Values Population**: If the values text area is empty, it will be populated with the chart's `values.yaml` content upon submission.
*   **API Endpoint**: A `/template` endpoint for programmatic access.

## Usage

To run the service locally, you need to have Go installed.

1.  Clone the repository:
    ```bash
    git clone <repository-url>
    cd helm-template-service
    ```

2.  Run the application:
    ```bash
    go run .
    ```

3.  Open your web browser and navigate to `http://localhost:8080`.

## Web Interface

The web interface provides a simple form with the following fields:

*   **Chart URL (.tgz)**: The URL to the `.tgz` Helm chart.
*   **Values (YAML)**: The values to override in the chart, in YAML format.

After submitting the form, the rendered output will be displayed below.

## API Usage

You can also use the service programmatically by sending a `POST` request to the `/template` endpoint.

### JSON Output

```bash
curl -X POST http://localhost:8080/template \
-H "Content-Type: application/json" \
-d '{
  "chartUrl": "https://charts.bitnami.com/bitnami/apache-8.9.1.tgz",
  "values": {
    "replicaCount": 2
  }
}'
```

### Raw Output

To get the raw Helm template output, you can use the `raw=true` query parameter. This is available for both `GET` and `POST` requests.

```bash
curl -X POST "http://localhost:8080/template?raw=true" \
-H "Content-Type: application/json" \
-d '{
  "chartUrl": "https://charts.bitnami.com/bitnami/apache-8.9.1.tgz",
  "values": {
    "replicaCount": 2
  }
}'
```

You can also use a `GET` request:

```bash
curl "http://localhost:8080/template?chartUrl=https://charts.bitnami.com/bitnami/apache-8.9.1.tgz&raw=true"
```

## Building with Podman

A `Containerfile` is provided to build a container image for the service.

1.  Build the image:
    ```bash
    podman build -t helm-template-service .
    ```

2.  Run the container:
    ```bash
    podman run -p 8080:8080 helm-template-service
    ```

The service will be available at `http://localhost:8080`.

## Development

This service was developed with the assistance of Roo Code and Google's Gemini.