import os
import time
import uuid

import pytest
import requests


BASE_URL = os.getenv("BASE_URL", "http://127.0.0.1:8000")
REQUEST_TIMEOUT = 5


@pytest.fixture(scope="module")
def user_data():
    return {
        "username": f"user_{uuid.uuid4()}",
        "password": "password228",
    }


@pytest.fixture(scope="module")
def auth_token(user_data):
    response = requests.post(
        f"{BASE_URL}/register",
        json=user_data,
        timeout=REQUEST_TIMEOUT,
    )
    assert response.status_code == 201, response.text

    response = requests.post(
        f"{BASE_URL}/login",
        json=user_data,
        timeout=REQUEST_TIMEOUT,
    )
    assert response.status_code == 200, response.text

    data = response.json()
    assert "token" in data

    return data["token"]


def test_register_user():
    user = {
        "username": f"user_{uuid.uuid4()}",
        "password": "password228",
    }

    response = requests.post(
        f"{BASE_URL}/register",
        json=user,
        timeout=REQUEST_TIMEOUT,
    )

    assert response.status_code == 201, response.text


def test_login_user(auth_token):
    assert auth_token


def get_code_processor_payload():
    return {
        "translator": "python3",
        "code": "print('Hello, stdout world!')",
    }


def create_task(auth_token):
    response = requests.post(
        f"{BASE_URL}/task",
        headers={"Authorization": f"Bearer {auth_token}"},
        json=get_code_processor_payload(),
        timeout=REQUEST_TIMEOUT,
    )

    assert response.status_code == 201, response.text

    data = response.json()
    assert "task_id" in data

    return data["task_id"]

def test_create_task(auth_token):
    task_id = create_task(auth_token)
    assert task_id

def test_task_status_and_result(auth_token):
    task_id = create_task(auth_token)

    headers = {"Authorization": f"Bearer {auth_token}"}

    status_url = f"{BASE_URL}/status/{task_id}"
    result_url = f"{BASE_URL}/result/{task_id}"

    deadline = time.monotonic() + 60

    while time.monotonic() < deadline:
        response = requests.get(
            status_url,
            headers=headers,
            timeout=REQUEST_TIMEOUT,
        )

        assert response.status_code == 200, response.text

        status = response.json()["status"]

        if status == "ready":
            break

        assert status == "in_progress", f"Unexpected status: {status}"

        time.sleep(2)
    else:
        pytest.fail("Task did not become ready within 60 seconds")

    response = requests.get(
        result_url,
        headers=headers,
        timeout=REQUEST_TIMEOUT,
    )

    assert response.status_code == 200, response.text
    assert response.json()["result"] == "Hello, stdout world!\n"


def test_task_not_found(auth_token):
    invalid_task_id = str(uuid.uuid4())

    headers = {"Authorization": f"Bearer {auth_token}"}

    response = requests.get(
        f"{BASE_URL}/status/{invalid_task_id}",
        headers=headers,
        timeout=REQUEST_TIMEOUT,
    )
    assert response.status_code == 404

    response = requests.get(
        f"{BASE_URL}/result/{invalid_task_id}",
        headers=headers,
        timeout=REQUEST_TIMEOUT,
    )
    assert response.status_code == 404


def test_unauthorized_access():
    invalid_task_id = str(uuid.uuid4())

    response = requests.post(
        f"{BASE_URL}/task",
        timeout=REQUEST_TIMEOUT,
    )
    assert response.status_code == 401

    response = requests.get(
        f"{BASE_URL}/status/{invalid_task_id}",
        timeout=REQUEST_TIMEOUT,
    )
    assert response.status_code == 401

    response = requests.get(
        f"{BASE_URL}/result/{invalid_task_id}",
        timeout=REQUEST_TIMEOUT,
    )
    assert response.status_code == 401