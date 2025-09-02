document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('template-form');
    const outputElement = document.getElementById('output');
    const copyButton = document.getElementById('copy-button');
    const downloadButton = document.getElementById('download-button');

    form.addEventListener('submit', async (event) => {
        event.preventDefault();

        const chartUrl = document.getElementById('chart-url').value;

        try {
            const url = new URL(chartUrl);
            if (!url.pathname.endsWith('.tgz')) {
                outputElement.textContent = 'Error: Chart URL must end with .tgz';
                return;
            }
        } catch (error) {
            outputElement.textContent = 'Error: Invalid Chart URL.';
            return;
        }

        const values = document.getElementById('values').value;

        let valuesJson = {};
        if (values.trim() !== '') {
            try {
                valuesJson = jsyaml.load(values);
            } catch (error) {
                outputElement.textContent = `Error parsing YAML: ${error.message}`;
                return;
            }
        }

        try {
            const response = await fetch('/template', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    chartUrl,
                    values: valuesJson,
                }),
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`);
            }

            const result = await response.json();
            outputElement.textContent = result.templates;
            hljs.highlightElement(outputElement);

            if (result.valuesExist && values.trim() === '') {
                document.getElementById('values').value = result.values;
            }
        } catch (error) {
            outputElement.textContent = `Error: ${error.message}`;
        }
    });

    copyButton.addEventListener('click', () => {
        navigator.clipboard.writeText(outputElement.textContent)
            .then(() => {
                alert('Copied to clipboard!');
            })
            .catch(err => {
                alert(`Failed to copy: ${err}`);
            });
    });

    downloadButton.addEventListener('click', () => {
        const blob = new Blob([outputElement.textContent], { type: 'text/yaml' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'rendered-manifests.yaml';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    });
});