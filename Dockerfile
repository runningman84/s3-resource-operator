# Use an official Python runtime as a parent image
FROM python:3.14-slim-bullseye

# Set the working directory in the container
WORKDIR /app

# Copy the requirements file into the container
COPY requirements.txt .

# Install any needed packages specified in requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy the application source code
COPY src/ ./src/

# Run the application when the container launches
CMD ["python", "-m", "src.main"]