from flask import Flask, request

app = Flask(__name__)

@app.route("/flask/test1", methods=["GET", "POST"])
def test1():
    return "body normal", 200, [("test-header", "hahaha")]

@app.route("/flask/test2", methods=["GET", "POST"])
def test2():
    return "body attack", 200, []

if __name__ == "__main__":
    app.run("0.0.0.0", 5000)
