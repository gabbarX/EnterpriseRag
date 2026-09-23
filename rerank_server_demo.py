import gc
import torch
import uvicorn
from fastapi import FastAPI
from pydantic import BaseModel, Field
from transformers import AutoModelForSequenceClassification, AutoTokenizer
from typing import List

# Enable CUDA debugging
# import os
# os.environ['CUDA_LAUNCH_BLOCKING']='1'

# --- 1. Request and response data structures for the API ---

class RerankRequest(BaseModel):
    query: str
    documents: List[str]

# The response structure used for testing exposes the field as "score"
class DocumentInfo(BaseModel):
    text: str

# TestRankResult replaces the original GoRankResult: the "relevance_score"
# field is renamed to "score".
class TestRankResult(BaseModel):
    index: int
    document: DocumentInfo
    score: float  # Key change: renamed from relevance_score to score

# Final response body: its "results" list holds TestRankResult entries
class TestFinalResponse(BaseModel):
    results: List[TestRankResult]


# --- 2. Load the model (once, at service startup) ---
print("Loading model, please wait...")
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
print(f"Device in use: {device}")
try:
    # Make sure this path is correct for your environment
    model_path = '/data1/home/lwx/work/Download/rerank_model_weight'
    tokenizer = AutoTokenizer.from_pretrained(model_path)
    model = AutoModelForSequenceClassification.from_pretrained(model_path)
    model.to(device)
    model.eval()
    print("Model loaded successfully.")
except Exception as e:
    print(f"Failed to load model: {e}")
    # In a test environment, exit rather than run a service that cannot work
    exit()

# --- 3. Create the FastAPI application ---
app = FastAPI(
    title="Reranker API (Test Version)",
    description="API service that returns a 'score' field, used to test Go client compatibility",
    version="1.0.2"
)

# --- 4. API endpoints ---
@app.post("/rerank", response_model=TestFinalResponse)
def rerank_endpoint(request: RerankRequest):
    pairs = [[request.query, doc] for doc in request.documents]

    with torch.no_grad():
        inputs = outputs = logits = None

        try:
            inputs = tokenizer(pairs, padding=True, truncation=True, return_tensors='pt', max_length=1024).to(device)
            outputs = model(**inputs, return_dict=True)
            logits = outputs.logits.view(-1, ).float()
            scores = torch.sigmoid(logits)
        finally:
            # Release GPU resources
            del inputs, outputs, logits
            gc.collect()

            if torch.cuda.is_available():
                torch.cuda.empty_cache()
            elif hasattr(torch, "mps") and torch.mps.is_available():
                torch.mps.empty_cache()


    # Build the results using the test response structure
    results = []
    for i, (text, score_val) in enumerate(zip(request.documents, scores)):
        doc_info = DocumentInfo(text=text)

        test_result = TestRankResult(
            index=i,
            document=doc_info,
            score=score_val.item(),
        )
        results.append(test_result)

    # Sort by score, descending
    sorted_results = sorted(results, key=lambda x: x.score, reverse=True)

    # FastAPI validates and serialises this dict against response_model
    # (TestFinalResponse), producing
    # {"results": [{"index": ..., "document": ..., "score": ...}]}
    return {"results": sorted_results}

@app.get("/")
def read_root():
    return {"status": "Reranker API (Test Version) is running"}

# --- 5. Start the service ---
if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
