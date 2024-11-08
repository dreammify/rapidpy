# 0
import requests
import flask
from io import BytesIO

if __name__ == '__main__':
    # Url for getting a random cat picture
    cataas_url = 'https://cataas.com/cat'
    # Fetch random cat picture
    cat_picture = requests.get(cataas_url).content

    # Create a Flask app
    app = flask.Flask(__name__)

    # Serve the random cat picture
    @app.route('/')
    def home():
        return flask.send_file(BytesIO(cat_picture), mimetype='image/jpeg')

    # Run the app
    app.run(host='0.0.0.0', port=45056)
