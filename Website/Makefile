.PHONY: test smoke up down

# Python unit tests
test:
	PYTHONPATH=Website python3 -m pytest -q Website/tests

# Build image and run container, then poll /health
smoke:
	docker build -t beacon-router:local Website \
	&& docker rm -f beacon-router-test >/dev/null 2>&1 || true \
	&& docker run --rm -d --name beacon-router-test -e PORT=8000 -p 8000:8000 beacon-router:local \
	    python3 -m uvicorn hybrid_router.main:app --host 0.0.0.0 --port 8000 --proxy-headers --forwarded-allow-ips "*" \
	&& for i in `seq 1 60`; do \
		if curl -fsS http://127.0.0.1:8000/health >/dev/null; then echo "Health OK"; break; fi; sleep 1; done \
	&& curl -s http://127.0.0.1:8000/health \
	&& docker rm -f beacon-router-test >/dev/null

up:
	docker run --rm -d --name beacon-router-dev -e PORT=8000 -p 8000:8000 beacon-router:local \
	    python3 -m uvicorn hybrid_router.main:app --host 0.0.0.0 --port 8000 --proxy-headers --forwarded-allow-ips "*"

down:
	docker rm -f beacon-router-dev >/dev/null 2>&1 || true
