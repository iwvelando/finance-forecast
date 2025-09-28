const form = document.getElementById("upload-form");
const fileInput = document.getElementById("config-file");
const runButton = document.getElementById("run-button");
const loadingIndicator = document.getElementById("loading-indicator");
const messageEl = document.getElementById("message");
const warningsEl = document.getElementById("warnings");
const resultsSection = document.getElementById("results-section");
const tableHead = document.querySelector("#results-table thead");
const tableBody = document.querySelector("#results-table tbody");
const downloadLink = document.getElementById("download-link");
const durationEl = document.getElementById("duration");

let currentObjectUrl = null;

form.addEventListener("submit", async (event) => {
    event.preventDefault();

    if (!fileInput.files || fileInput.files.length === 0) {
        showMessage("Please select a YAML file to continue.", "error");
        return;
    }

    toggleLoading(true);
    clearResults();

    try {
        const formData = new FormData();
        formData.append("file", fileInput.files[0]);

        const response = await fetch("/api/forecast", {
            method: "POST",
            body: formData,
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || "Unable to process forecast");
        }

        renderResults(data);
        showMessage("Forecast completed successfully.", "success");
    } catch (error) {
        console.error("Forecast request failed", error);
        showMessage(error.message, "error");
    } finally {
        toggleLoading(false);
    }
});

function toggleLoading(isLoading) {
    runButton.disabled = isLoading;
    loadingIndicator.classList.toggle("hidden", !isLoading);
}

function showMessage(message, type) {
    messageEl.textContent = message;
    messageEl.className = type ? `message ${type}` : "message";
}

function clearResults() {
    warningsEl.textContent = "";
    warningsEl.classList.add("hidden");
    tableHead.innerHTML = "";
    tableBody.innerHTML = "";
    resultsSection.classList.add("hidden");
    downloadLink.classList.add("hidden");
    durationEl.textContent = "";

    if (currentObjectUrl) {
        URL.revokeObjectURL(currentObjectUrl);
        currentObjectUrl = null;
    }
}

function renderResults(data) {
    if (!data || !Array.isArray(data.scenarios) || !Array.isArray(data.rows)) {
        throw new Error("Malformed response received from server");
    }

    renderWarnings(data.warnings);
    renderTable(data.scenarios, data.rows);
    prepareDownload(data.csv);

    if (data.duration) {
        durationEl.textContent = `Computed in ${data.duration}`;
    }

    resultsSection.classList.remove("hidden");
}

function renderWarnings(warnings) {
    if (!warnings || warnings.length === 0) {
        return;
    }

    warningsEl.innerHTML = `<strong>Warnings:</strong><ul>${warnings
        .map((warning) => `<li>${escapeHtml(warning)}</li>`)
        .join("")}</ul>`;
    warningsEl.classList.remove("hidden");
}

function renderTable(scenarios, rows) {
    const headRow = document.createElement("tr");
    headRow.appendChild(createHeaderCell("Date"));
    scenarios.forEach((scenario) => {
        const th = createHeaderCell(scenario);
        th.colSpan = 2;
        headRow.appendChild(th);
    });
    tableHead.appendChild(headRow);

    const subHeadRow = document.createElement("tr");
    subHeadRow.appendChild(createHeaderCell("", "subhead"));
    scenarios.forEach(() => {
        subHeadRow.appendChild(createHeaderCell("Amount", "subhead"));
        subHeadRow.appendChild(createHeaderCell("Notes", "subhead"));
    });
    tableHead.appendChild(subHeadRow);

    const currencyFormatter = new Intl.NumberFormat(undefined, {
        style: "currency",
        currency: "USD",
        minimumFractionDigits: 2,
    });

    rows.forEach((row) => {
        const tr = document.createElement("tr");
        tr.appendChild(createCell(row.date));

        row.values.forEach((value) => {
            const amountText = typeof value.amount === "number"
                ? currencyFormatter.format(value.amount)
                : "—";
            const amountCell = createCell(amountText, "amount-cell");
            tr.appendChild(amountCell);

            const notesCell = createCell(formatNotes(value.notes));
            tr.appendChild(notesCell);
        });

        tableBody.appendChild(tr);
    });
}

function prepareDownload(csvContent) {
    if (!csvContent) {
        return;
    }

    const blob = new Blob([csvContent], { type: "text/csv" });
    currentObjectUrl = URL.createObjectURL(blob);
    downloadLink.href = currentObjectUrl;
    downloadLink.download = `forecast-${new Date().toISOString().split("T")[0]}.csv`;
    downloadLink.classList.remove("hidden");
}

function createHeaderCell(text, className = "") {
    const th = document.createElement("th");
    th.textContent = text;
    if (className) th.classList.add(className);
    return th;
}

function createCell(content, className = "") {
    const td = document.createElement("td");
    td.innerHTML = content;
    if (className) td.classList.add(className);
    return td;
}

function formatNotes(notes) {
    if (!Array.isArray(notes) || notes.length === 0) {
        return "<span class=\"muted-text\">—</span>";
    }

    return `<ul class="note-list">${notes
        .map((note) => `<li>${escapeHtml(note)}</li>`)
        .join("")}</ul>`;
}

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}
