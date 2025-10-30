FROM python:3.9
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY k8s/demo-app/demo-app.py ./demo-app.py
CMD ["python", "demo-app.py"]
