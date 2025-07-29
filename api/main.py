from fastapi import FastAPI
from orchestrator.main import run_job, get_status

app = FastAPI()

@app.post('/schedule')
def schedule_job(prompt: str):
    job_id = run_job(prompt)
    return {"job_id": job_id}

@app.get('/status/${job_id}')
def status(job_id: str):
    return get_status(job_id)