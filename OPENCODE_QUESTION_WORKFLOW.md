# Opencode Question & Feedback Workflow

This document details the internal workflow of the Opencode frontend for handling user feedback/questions and provides a guide for implementing this functionality in external skills (e.g., Python scripts).

## 1. Overview

The "Question/Feedback" mechanism allows the backend agent to pause execution and request input from the user (e.g., clarifying requirements, selecting a path, or confirming an action). The frontend displays these requests interactively, and the agent resumes once a reply is received.

## 2. Internal Frontend Workflow

The Opencode web client (`opencode-dev/packages/app`) handles this workflow using a combination of **Server-Sent Events (SSE)** for real-time updates and standard HTTP REST endpoints for actions.

### Step 1: Listening (Real-time Events)
The frontend maintains a persistent connection to the global event stream.
- **Endpoint**: `GET /global/event`
- **Mechanism**: Server-Sent Events (SSE)
- **Event Type**: `question.asked`
- **Handler**: `packages/app/src/context/global-sync/event-reducer.ts`

### Step 2: State Update & UI Rendering
1. When a `question.asked` event is received, the payload (a `QuestionRequest` object) is stored in the local sync state (`sync.data.question`).
2. The `SessionPromptDock` component observes this state.
3. If a request exists, it renders the `QuestionDock` component (`packages/app/src/components/question-dock.tsx`), displaying the options to the user.

### Step 3: User Action & Reply
When the user selects an option or types a custom answer:
1. The component constructs an answer payload.
2. It calls the `POST /question/{requestID}/reply` endpoint.
3. The backend receives the answer, resolves the promise waiting on the agent side, and the agent resumes execution.

---

## 3. API Integration Guide (for External Skills)

For external scripts (like `opencode-web-skill`), implementing a full SSE listener can be complex. The recommended approach is to use the **Polling API** to check for pending questions.

### Endpoints

#### 1. List Pending Questions
Retrieves all currently active question requests that are waiting for user input.

- **Method**: `GET`
- **URL**: `/question`
- **Headers**:
  - `x-opencode-directory`: `/path/to/project`

**Response Structure (Simplified):**
```json
{
  "data": [
    {
      "id": "req_12345",
      "sessionID": "ses_abcde",
      "questions": [
        {
          "question": "Which file do you want to edit?",
          "options": [
            { "label": "main.py", "description": "Entry point" },
            { "label": "utils.py", "description": "Utilities" }
          ],
          "multiple": false
        }
      ]
    }
  ]
}
```

#### 2. Submit Answer
Submits the selected option(s) or text back to the agent.

- **Method**: `POST`
- **URL**: `/question/{requestID}/reply`
- **Headers**:
  - `Content-Type`: `application/json`
  - `x-opencode-directory`: `/path/to/project`
- **Body**:
  ```json
  {
    "answers": [
      ["selected_value"] 
    ]
  }
  ```
  *Note: `answers` is an array of arrays. The outer array corresponds to the list of questions associated with the request (usually just one).*

#### 3. Reject/Dismiss Request
Cancels the request, potentially causing the agent to abort or retry.

- **Method**: `POST`
- **URL**: `/question/{requestID}/reject`

---

## 4. Python Implementation Example

The following script demonstrates how to poll for pending questions and automatically answer them (or prompt the user in the terminal).

```python
import requests
import time
import json

# Configuration
BASE_URL = "http://127.0.0.1:4096"
PROJECT_DIR = "/Users/nguyenminhkha/All/Code/ns-projects/ns-kb"  # Update this

HEADERS = {
    "x-opencode-directory": PROJECT_DIR,
    "Content-Type": "application/json",
    # Add other headers as needed (e.g., User-Agent)
}

def check_and_answer_questions():
    try:
        # 1. List pending questions
        resp = requests.get(f"{BASE_URL}/question", headers=HEADERS)
        resp.raise_for_status()
        
        data = resp.json().get('data', [])
        
        if not data:
            print("[*] No pending questions.")
            return

        for req in data:
            request_id = req['id']
            # Assuming single question per request for simplicity
            question_obj = req['questions'][0]
            question_text = question_obj['question']
            options = question_obj.get('options', [])
            
            print(f"\n[?] Question: {question_text}")
            
            # --- Logic to determine answer ---
            # In a real script, this might be automated or passed from args.
            # Here, we default to the first available option.
            
            answer = []
            if options:
                first_option = options[0]['label']
                print(f"    -> Auto-selecting first option: {first_option}")
                answer = [first_option]
            else:
                # Handle text input case
                print("    -> Auto-replying with default text.")
                answer = ["Default Answer"]
            
            # A request can have multiple questions, so we wrap our single answer in a list
            # The structure is: answers[question_index] = [selected_option_1, selected_option_2...]
            payload = {
                "answers": [answer]
            }

            # 2. Submit Reply
            print(f"    -> Sending reply to {request_id}...")
            reply_resp = requests.post(
                f"{BASE_URL}/question/{request_id}/reply", 
                json=payload, 
                headers=HEADERS
            )
            
            if reply_resp.status_code == 200:
                print("    [+] Successfully replied!")
            else:
                print(f"    [-] Failed to reply: {reply_resp.text}")

    except Exception as e:
        print(f"[!] Error checking questions: {e}")

if __name__ == "__main__":
    print("Starting Question Poller...")
    while True:
        check_and_answer_questions()
        time.sleep(2)
```
