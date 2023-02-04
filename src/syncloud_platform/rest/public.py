import sys
import traceback

import requests
from flask import jsonify, request, redirect, Flask, Response
from flask_login import LoginManager, login_user, logout_user, current_user, login_required
from syncloud_platform.injector import get_injector
from syncloud_platform.rest.backend_proxy import backend_request
from syncloud_platform.rest.flask_decorators import fail_if_not_activated, fail_if_activated
from syncloud_platform.rest.model.flask_user import FlaskUser
from syncloud_platform.rest.model.user import User
from syncloud_platform.rest.service_exception import ServiceException
from syncloudlib.error import PassthroughJsonError
from syncloudlib.json import convertible
from syncloudlib.logger import get_logger

injector = get_injector()
public = injector.public

app = Flask(__name__)
app.config['SECRET_KEY'] = public.user_platform_config.get_web_secret_key()
login_manager = LoginManager()
login_manager.init_app(app)
log = get_logger('ldap')


@login_manager.unauthorized_handler
def _callback():
    log.warn('Unauthorised handler 401')
    return 'Unauthorised', 401


@login_manager.user_loader
def load_user(user_id):
    log.info('loading user {0}'.format(user_id))
    return FlaskUser(User(user))


@app.route("/rest/login", methods=["POST"])
@fail_if_not_activated
def login():
    request_json = request.json
    if 'username' in request_json and 'password' in request_json:
        try:
            injector.ldap_auth.authenticate(request_json['username'], request_json['password'])
            user_flask = FlaskUser(User(request_json['username']))
            log.info('login user {0}'.format(user_flask.user.name))
            login_user(user_flask, remember=False)
            # next_url = request.get('next_url', '/')
            return redirect("/")
        except Exception as e:
            traceback.print_exc(file=sys.stdout)
            return jsonify(message=str(e)), 400

    return jsonify(message='missing name or password'), 400


@app.route("/rest/logout", methods=["POST"])
@fail_if_not_activated
@login_required
def logout():
    log.info('logout user {0}'.format(current_user.user.name))
    logout_user()
    return 'User logged out', 200


@app.route("/rest/user", methods=["GET"])
@fail_if_not_activated
@login_required
def user():
    log.info('current user {0}'.format(current_user.user.name))
    return jsonify(convertible.to_dict(current_user.user)), 200


@app.route("/rest/send_log", methods=["POST"])
@fail_if_not_activated
@login_required
def send_log():
    include_support = request.args['include_support'] == 'true'
    public.send_logs(include_support)
    return jsonify(success=True), 200


@app.route("/rest/app_image", methods=["GET"])
@fail_if_not_activated
@login_required
def app_image():
    channel = request.args['channel']
    app = request.args['app']
    r = requests.get('http://apps.syncloud.org/releases/{0}/images/{1}-128.png'.format(channel, app), stream=True)
    return Response(r.iter_content(chunk_size=10 * 1024),
                    content_type=r.headers['Content-Type'])


@app.route("/rest/backup/list", methods=["GET"])
@app.route("/rest/backup/create", methods=["POST"])
@app.route("/rest/backup/restore", methods=["POST"])
@app.route("/rest/backup/remove", methods=["POST"])
@app.route("/rest/backup/auto", methods=["GET", "POST"])
@app.route("/rest/installer/upgrade", methods=["POST"])
@app.route("/rest/installer/version", methods=["GET"])
@app.route("/rest/installer/status", methods=["GET"])
@app.route("/rest/job/status", methods=["GET"])
@app.route("/rest/storage/deactivate", methods=["POST"])
@app.route("/rest/storage/activate/partition", methods=["POST"])
@app.route("/rest/storage/activate/disk", methods=["POST"])
@app.route("/rest/storage/boot_extend", methods=["POST"])
@app.route("/rest/storage/boot/disk", methods=["GET"])
@app.route("/rest/storage/disks", methods=["GET"])
@app.route("/rest/storage/error/last", methods=["GET"])
@app.route("/rest/storage/error/clear", methods=["POST"])
@app.route("/rest/event/trigger", methods=["POST"])
@app.route("/rest/certificate", methods=["GET"])
@app.route("/rest/certificate/log", methods=["GET"])
@app.route("/rest/access", methods=["GET", "POST"])
@app.route("/rest/apps/available", methods=["GET"])
@app.route("/rest/apps/installed", methods=["GET"])
@app.route("/rest/logs", methods=["GET"])
@app.route("/rest/app", methods=["GET"])
@app.route("/rest/device/url", methods=["GET"])
@app.route("/rest/deactivate", methods=["POST"])
@app.route("/rest/app/install", methods=["POST"])
@app.route("/rest/app/remove", methods=["POST"])
@app.route("/rest/app/upgrade", methods=["POST"])
@app.route("/rest/restart", methods=["POST"])
@app.route("/rest/shutdown", methods=["POST"])
@app.route("/rest/network/interfaces", methods=["GET"])
@fail_if_not_activated
@login_required
def backend_proxy_activated():
    response = backend_request(request.method, request.full_path.replace("/rest", "", 1), request.json)
    headers = {'Content-Type': response.headers['Content-Type']}
    return response.text, response.status_code, headers


@app.route("/rest/redirect/domain/availability", methods=["POST"])
@app.route("/rest/redirect_info", methods=["GET"])
@app.route("/rest/activate/managed", methods=["POST"])
@app.route("/rest/activate/custom", methods=["POST"])
@fail_if_activated
def backend_proxy_not_activated():
    response = backend_request(request.method, request.full_path.replace("/rest", "", 1), request.json)
    return response.text, response.status_code


@app.route("/rest/activation/status", methods=["GET"])
def backend_proxy():
    response = backend_request(request.method, request.full_path.replace("/rest", "", 1), request.json)
    return response.text, response.status_code


@app.route("/rest/id", methods=["GET"])
def identification():
    response = backend_request("GET", "/id", None)
    return response.text, response.status_code


@app.errorhandler(Exception)
def handle_exception(error):
    print('-' * 60)
    traceback.print_exc(file=sys.stdout)
    print('-' * 60)
    status_code = 500

    if isinstance(error, PassthroughJsonError):
        return Response(error.json, status=status_code, mimetype='application/json')

    if isinstance(error, ServiceException):
        status_code = 200

    response = jsonify(success=False, message=str(error))
    return response, status_code


if __name__ == '__main__':
    app.run(host='0.0.0.0', debug=True, port=5001)
