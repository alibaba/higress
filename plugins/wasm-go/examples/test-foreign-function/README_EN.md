## Introduction

This example is used to show how to add body data when response has no body.

**Attention**
1. Should return `types.ActionPause` on response header phase.
2. `Endstream` should be set with `true`


A Flask app demo:

```python
import os
from flask import Flask, request, Response

app = Flask(__name__)

@app.route('/test', methods=['GET', 'POST'])
def print_request():
    return Response(status=200)

if __name__ == '__main__':
    app.run("0.0.0.0", 5000, debug=False)
```