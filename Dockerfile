FROM python:3
WORKDIR /usr/src/app
COPY pyproject.toml ./
RUN pip3 install poetry
RUN poetry config virtualenvs.create false
RUN poetry install --no-dev
COPY emu2mqtt.py ./
CMD [ "python", "./emu2mqtt.py"]