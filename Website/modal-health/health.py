import time
import modal

image = modal.Image.debian_slim().pip_install("fastapi")
app = modal.App("project-beacon-modal-health")

@app.function()
def ping() -> dict:
    """Callable function (non-HTTP) for basic ping."""
    return {"status": "healthy", "timestamp": time.time(), "service": "modal-health-fn"}

@app.function(image=image)
@modal.fastapi_endpoint()
def http_health() -> dict:
    """HTTP endpoint that always reports healthy.

    After deploy, this will be available at a URL like:
    https://<username>--project-beacon-modal-health-http-health.modal.run
    """
    return {"status": "healthy", "timestamp": time.time(), "service": "modal-health-http"}
