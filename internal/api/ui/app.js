let selectedJob = null;
let refreshing = false;

const examples = {
    success: `name: Forge Validation

steps:
  - name: Environment Check
    image: alpine:3
    commands:
      - echo "Forge execution started"
      - uname -a

  - name: Run Tests
    image: alpine:3
    commands:
      - test "$((5 + 5))" -eq 10
      - echo "All tests passed"

  - name: Build Artifact
    image: alpine:3
    commands:
      - echo "Building application"
      - echo "Build completed successfully"
`,
    failure: `name: Failed Build Demo

steps:
  - name: Initialize
    image: alpine:3
    commands:
      - echo "Preparing build"

  - name: Failed Test
    image: alpine:3
    commands:
      - echo "Test failure detected"
      - exit 1

  - name: Publish
    image: alpine:3
    commands:
      - echo "This must not execute"
`,
    concurrent: `name: Concurrent Worker Demo

steps:
  - name: Background Task
    image: alpine:3
    commands:
      - echo "Worker started"
      - sleep 10
      - echo "Worker finished"
`
};

function loadExample(name) {
    document.getElementById("pipeline").value = examples[name];
}

async function request(url, options = {}) {
    const response = await fetch(url, options);
    const data = await response.json();

    if (!response.ok) {
        throw new Error(data.error || "Request failed");
    }

    return data;
}

function statusColor(status) {
    return {
        succeeded: "green",
        failed: "red",
        running: "blue",
        queued: "amber"
    }[status] || "";
}

function duration(job) {
    if (!job.started_at) return "—";

    const start = Date.parse(job.started_at);

    const end = job.ended_at
        ? Date.parse(job.ended_at)
        : Date.now();

    if (!Number.isFinite(start) || !Number.isFinite(end)) {
        return "—";
    }

    return Math.max(0, (end - start) / 1000).toFixed(1) + "s";
}

function updateMetrics(jobs) {
    document.getElementById("total").textContent = jobs.length;

    const metrics = {
        success: "succeeded",
        failed: "failed",
        running: "running",
        queued: "queued"
    };

    for (const [id, status] of Object.entries(metrics)) {
        document.getElementById(id).textContent =
            jobs.filter(job => job.status === status).length;
    }
}

function renderJobs(jobs) {
    const table = document.getElementById("jobs");
    table.replaceChildren();

    for (const job of jobs) {
        const row = document.createElement("tr");

        if (job.id === selectedJob) {
            row.classList.add("selected");
        }

        row.onclick = () => {
            selectedJob = job.id;
            refresh();
        };

        const values = [
            "#" + job.id,
            job.name,
            job.status.toUpperCase(),
            new Date(job.created_at).toLocaleString(),
            duration(job)
        ];

        values.forEach((value, index) => {
            const cell = document.createElement("td");

            if (index === 2) {
                const badge = document.createElement("span");
                badge.className = "status " + statusColor(job.status);
                badge.textContent = value;
                cell.appendChild(badge);
            } else {
                cell.textContent = value;
            }

            row.appendChild(cell);
        });

        table.appendChild(row);
    }
}

async function loadJob(id) {
    const job = await request("/api/jobs/" + id);

    if (selectedJob !== id) return;

    document.getElementById("selected-job").textContent =
        "BUILD #" + job.id;

    document.getElementById("job-status").textContent =
        job.status.toUpperCase() + " · " + duration(job);

    const output = document.getElementById("logs");

    const atBottom =
        output.scrollHeight - output.scrollTop - output.clientHeight < 30;

    output.textContent =
        (job.logs || "Waiting for execution logs...") +
        (job.error ? "\nERROR: " + job.error : "");

    if (atBottom) output.scrollTop = output.scrollHeight;
}

async function refresh() {
    if (refreshing) return;
    refreshing = true;

    const connection = document.getElementById("connection");

    try {
        const jobs = await request("/api/jobs");

        connection.className = "connection green";
        connection.textContent = "● Engine Online";

        if (selectedJob === null && jobs.length) {
            selectedJob = jobs[0].id;
        }

        updateMetrics(jobs);
        renderJobs(jobs);

        if (selectedJob !== null) {
            await loadJob(selectedJob);
        }

    } catch (error) {
        connection.className = "connection red";
        connection.textContent = "● Connection Error";
    } finally {
        refreshing = false;
    }
}

async function submitPipeline() {
    const button = document.getElementById("run-button");
    const message = document.getElementById("message");

    button.disabled = true;
    message.textContent = "Submitting pipeline...";

    try {
        const job = await request("/api/jobs", {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify({
                pipeline: document.getElementById("pipeline").value
            })
        });

        selectedJob = job.id;

        message.textContent = "Build #" + job.id + " queued";

        await refresh();

    } catch (error) {
        message.textContent = error.message;
    } finally {
        button.disabled = false;
    }
}

async function copyLogs() {
    const message = document.getElementById("message");

    try {
        await navigator.clipboard.writeText(
            document.getElementById("logs").textContent
        );

        message.textContent = "Logs copied";
    } catch {
        message.textContent = "Select and copy the logs manually";
    }
}

loadExample("success");
refresh();
setInterval(refresh, 2000);
