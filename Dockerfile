FROM python:3
WORKDIR /usr/src/app
COPY ./ ./
RUN pip3 install poetry
RUN poetry install
CMD [ "poetry", "run", "emu2mqtt"]