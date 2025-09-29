const messageEl = document.getElementById("message");
const warningsEl = document.getElementById("warnings");
const resultsPanel = document.getElementById("results-section");
const configPanel = document.getElementById("config-panel");
const tableHead = document.querySelector("#results-table thead");
const tableBody = document.querySelector("#results-table tbody");
const downloadLink = document.getElementById("download-link");
const durationEl = document.getElementById("duration");
const configEditorRoot = document.getElementById("config-editor");
const uploadConfigInput = document.getElementById("upload-config-input");
const uploadConfigButton = document.getElementById("upload-config-button");
const runForecastButton = document.getElementById("run-forecast-button");
const downloadConfigButton = document.getElementById("download-config-button");
const resetConfigButton = document.getElementById("reset-config-button");
const editorLoading = document.getElementById("editor-loading");
const tablistContainer = document.querySelector(".tablist-container");
if (configPanel) {
	configPanel.classList.add("sticky-headers");
}

const rootStyle = document.documentElement.style;

const tabButtons = Array.from(document.querySelectorAll(".tab-button"));
const tabPanels = {
	results: resultsPanel,
	config: configPanel,
};
const resultsTabButton = document.getElementById("tab-results");

let activeTab = "config";
let dataAvailable = false;
let currentObjectUrl = null;
let configDownloadUrl = null;
let currentConfig = null;
let hiddenLogging = null;
let latestConfigYaml = "";
let defaultConfigInitialized = false;
let isEditorLoading = false;
let registeredInputs = [];

const MONTH_PATTERN = /^\d{4}-(0[1-9]|1[0-2])$/;

tabButtons.forEach((button) => {
	button.addEventListener("click", () => {
		const tab = button.dataset.tab;
		if (tab) {
			switchTab(tab);
		}
	});
});

if (uploadConfigButton) {
	uploadConfigButton.addEventListener("click", () => {
		if (uploadConfigInput) {
			uploadConfigInput.click();
		}
	});
}

if (uploadConfigInput) {
	uploadConfigInput.addEventListener("change", handleConfigFileSelection);
}

runForecastButton.addEventListener("click", handleRunForecast);
downloadConfigButton.addEventListener("click", downloadCurrentConfig);
resetConfigButton.addEventListener("click", handleResetConfig);

const updateStickyMetrics = () => {
	if (tablistContainer) {
		rootStyle.setProperty("--workspace-sticky-offset", `${tablistContainer.offsetHeight}px`);
	} else {
		rootStyle.setProperty("--workspace-sticky-offset", "0px");
	}

	const editorToolbar = document.querySelector(".editor-toolbar");
	if (editorToolbar) {
		const styles = window.getComputedStyle(editorToolbar);
		const marginBottom = parseFloat(styles.marginBottom || "0") || 0;
		rootStyle.setProperty("--config-toolbar-offset", `${editorToolbar.offsetHeight + marginBottom}px`);
	} else {
		rootStyle.setProperty("--config-toolbar-offset", "0px");
	}
};

updateStickyMetrics();
window.addEventListener("resize", updateStickyMetrics);
window.addEventListener("load", updateStickyMetrics);

initializeWorkspace();

function toggleEditorLoading(isLoading) {
	isEditorLoading = isLoading;
	if (uploadConfigButton) {
		uploadConfigButton.disabled = isLoading;
	}
	resetConfigButton.disabled = isLoading;
	editorLoading.classList.toggle("hidden", !isLoading);
	updateEditorActionsState();
	updateStickyMetrics();
}

function handleConfigFileSelection(event) {
	const input = event.target;
	const file = input && input.files && input.files[0] ? input.files[0] : null;
	if (!file) {
		return;
	}

	showMessage("", null);
	runForecastFromFile(file).finally(() => {
		input.value = "";
	});
}

async function runForecastFromFile(file) {
	toggleEditorLoading(true);
	clearResultsView();

	try {
		const formData = new FormData();
		formData.append("file", file);

		const response = await fetch("/api/forecast", {
			method: "POST",
			body: formData,
		});

		const data = await response.json();

		if (!response.ok) {
			throw new Error(data.error || "Unable to process forecast");
		}

		processForecastResponse(data, "Forecast completed successfully.");
	} catch (error) {
		console.error("Forecast request failed", error);
		showMessage(error.message, "error");
	} finally {
		toggleEditorLoading(false);
	}
}

function processForecastResponse(data, successMessage, options = {}) {
	const { switchToResults = true } = options;
	renderResults(data);
	updateConfigState(data);
	setDataAvailability(true);
	if (switchToResults) {
		switchTab("results");
	}
	showMessage(successMessage, "success");
}

function showMessage(message, type) {
	if (!message) {
		messageEl.textContent = "";
		messageEl.className = "message hidden";
		return;
	}

	messageEl.textContent = message;
	messageEl.className = type ? `message ${type}` : "message";
}

function clearResultsView() {
	warningsEl.textContent = "";
	warningsEl.classList.add("hidden");
	tableHead.innerHTML = "";
	tableBody.innerHTML = "";
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

	clearResultsView();
	tableHead.innerHTML = "";
	tableBody.innerHTML = "";

	renderWarnings(data.warnings);
	renderTable(data.scenarios, data.rows);
	prepareDownload(data.csv);

	if (data.duration) {
		durationEl.textContent = `Computed in ${data.duration}`;
	}

	updateStickyMetrics();
}

function renderWarnings(warnings) {
	if (!warnings || warnings.length === 0) {
		warningsEl.classList.add("hidden");
		return;
	}

	warningsEl.innerHTML = `<strong>Warnings:</strong><ul>${warnings
		.map((warning) => `<li>${escapeHtml(warning)}</li>`)
		.join("")}</ul>`;
	warningsEl.classList.remove("hidden");
}

function renderTable(scenarios, rows) {
	const headRow = document.createElement("tr");
	headRow.classList.add("primary-header-row");
	headRow.appendChild(createHeaderCell("Date"));
	scenarios.forEach((scenario) => {
		const th = createHeaderCell(scenario);
		th.colSpan = 2;
		headRow.appendChild(th);
	});
	tableHead.appendChild(headRow);

	const subHeadRow = document.createElement("tr");
	subHeadRow.classList.add("secondary-header-row");
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

	if (currentObjectUrl) {
		URL.revokeObjectURL(currentObjectUrl);
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

function updateConfigState(data) {
	const rawConfig = data && data.config ? data.config : null;

	if (!rawConfig) {
		currentConfig = createInitialConfig();
		hiddenLogging = getDefaultLoggingConfig();
		latestConfigYaml = "";
		renderConfigEditor();
		setDataAvailability(false);
		return;
	}

	const prepared = prepareConfigForEditing(rawConfig);
	currentConfig = prepared.config;
	hiddenLogging = prepared.logging || getDefaultLoggingConfig();
	if (!currentConfig.output || typeof currentConfig.output !== "object") {
		currentConfig.output = { format: "pretty" };
	} else if (!currentConfig.output.format) {
		currentConfig.output.format = "pretty";
	}
	latestConfigYaml = typeof data.configYaml === "string" ? data.configYaml : "";

	renderConfigEditor();
}

function prepareConfigForEditing(rawConfig) {
	const cloned = cloneDeep(rawConfig) || {};

	let loggingConfig = null;
	if (Object.prototype.hasOwnProperty.call(cloned, "logging")) {
		loggingConfig = cloneDeep(cloned.logging);
		delete cloned.logging;
	}

	if (!cloned.common || typeof cloned.common !== "object") {
		cloned.common = {};
	}

	const common = cloned.common;
	common.events = Array.isArray(common.events) ? common.events.map(normalizeEvent) : [];
	common.loans = Array.isArray(common.loans) ? common.loans.map(normalizeLoan) : [];

	cloned.scenarios = Array.isArray(cloned.scenarios)
		? cloned.scenarios.map(normalizeScenario)
		: [];

	if (!cloned.output || typeof cloned.output !== "object") {
		cloned.output = { format: "pretty" };
	} else if (!cloned.output.format) {
		cloned.output.format = "pretty";
	}

	return {
		config: cloned,
		logging: loggingConfig,
	};
}

function normalizeEvent(event) {
	const normalized = cloneDeep(event) || {};
	delete normalized.dateList;
	delete normalized.DateList;
	return normalized;
}

function normalizeLoan(loan) {
	const normalized = cloneDeep(loan) || {};
	delete normalized.amortizationSchedule;
	delete normalized.AmortizationSchedule;
	normalized.extraPrincipalPayments = Array.isArray(normalized.extraPrincipalPayments)
		? normalized.extraPrincipalPayments.map(normalizeEvent)
		: [];
	return normalized;
}

function normalizeScenario(scenario) {
	const normalized = cloneDeep(scenario) || {};
	normalized.events = Array.isArray(normalized.events)
		? normalized.events.map(normalizeEvent)
		: [];
	normalized.loans = Array.isArray(normalized.loans)
		? normalized.loans.map(normalizeLoan)
		: [];
	if (typeof normalized.active !== "boolean") {
		normalized.active = Boolean(normalized.active);
	}
	return normalized;
}

function renderConfigEditor() {
	configEditorRoot.innerHTML = "";
	registeredInputs = [];

	if (!currentConfig) {
		currentConfig = createInitialConfig();
	}

	const simulationSection = createSection("Simulation", "Control global simulation behavior.");
	const simGrid = document.createElement("div");
	simGrid.className = "editor-grid";
	simGrid.appendChild(createInputField({
		label: "Start date (YYYY-MM)",
		path: "startDate",
		value: currentConfig.startDate ?? "",
		inputType: "month",
		tooltip: "First month of the simulation in YYYY-MM format.",
		validation: { type: "month", required: true },
		maxLength: 7,
	}));
	simulationSection.body.appendChild(simGrid);
	configEditorRoot.appendChild(simulationSection.section);

	const commonSection = createSection("Common settings", "Shared events and loans applied to every scenario.");
	const commonGrid = document.createElement("div");
	commonGrid.className = "editor-grid";
	commonGrid.appendChild(createInputField({
		label: "Starting value",
		path: "common.startingValue",
		value: currentConfig.common.startingValue ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Balance at the end of the start month. Use positive values for assets and negative values for debt.",
		validation: { type: "number" },
	}));
	commonGrid.appendChild(createInputField({
		label: "Death date (YYYY-MM)",
		path: "common.deathDate",
		value: currentConfig.common.deathDate ?? "",
		inputType: "month",
		tooltip: "Month when the simulation ends (YYYY-MM).",
		validation: { type: "month", required: true },
		maxLength: 7,
	}));
	commonSection.body.appendChild(commonGrid);
	commonSection.body.appendChild(createEventCollection(currentConfig.common.events, "common.events", {
		heading: "Common events",
		titlePrefix: "Event",
		addLabel: "Add common event",
		headingClass: "sticky-heading",
	}));
	commonSection.body.appendChild(createLoanCollection(currentConfig.common.loans, "common.loans", {
		heading: "Common loans",
		addLabel: "Add common loan",
	}));
	configEditorRoot.appendChild(commonSection.section);

	const scenariosSection = createSection("Scenarios", "Create alternative projections with unique events and loans.");
	const scenariosContainer = document.createElement("div");
	scenariosContainer.className = "editor-collection";

	currentConfig.scenarios.forEach((scenario, index) => {
		const card = createScenarioCard(scenario, index);
		scenariosContainer.appendChild(card);
	});

	scenariosSection.body.appendChild(scenariosContainer);
	const scenarioActions = document.createElement("div");
	scenarioActions.className = "collection-actions";
	const addScenarioButton = document.createElement("button");
	addScenarioButton.type = "button";
	addScenarioButton.textContent = "Add scenario";
	addScenarioButton.addEventListener("click", () => {
		currentConfig.scenarios.push(createEmptyScenario());
		renderConfigEditor();
		switchTab("config");
	});
	scenarioActions.appendChild(addScenarioButton);
	scenariosSection.body.appendChild(scenarioActions);
	configEditorRoot.appendChild(scenariosSection.section);

	updateStickyMetrics();
	validateEditorForm();
}

function createSection(title, description) {
	const section = document.createElement("section");
	section.className = "editor-section";

	const header = document.createElement("div");
	header.className = "editor-section-header";
	const heading = document.createElement("h3");
	heading.textContent = title;
	header.appendChild(heading);
	section.appendChild(header);

	if (description) {
		const desc = document.createElement("p");
		desc.className = "description";
		desc.textContent = description;
		section.appendChild(desc);
	}

	const body = document.createElement("div");
	body.className = "editor-section-body";
	section.appendChild(body);

	return { section, body };
}

function createScenarioCard(scenario, index) {
	const card = document.createElement("div");
	card.className = "editor-card";

	const { header, title } = createCardHeader(
		scenario.name || `Scenario ${index + 1}`,
		currentConfig.scenarios.length > 1
			? () => {
				  currentConfig.scenarios.splice(index, 1);
				  renderConfigEditor();
			  }
		: null,
		"Remove scenario",
		{ extraClass: "scenario-card-header" },
	);

	card.appendChild(header);

	const grid = document.createElement("div");
	grid.className = "editor-grid";
	grid.appendChild(createInputField({
		label: "Scenario name",
		path: `scenarios[${index}].name`,
		value: scenario.name ?? "",
		inputType: "text",
		placeholder: "e.g., Base case",
		tooltip: "Display name for this scenario in tables and charts.",
		validation: { type: "text", maxLength: 120 },
		maxLength: 120,
		onChange: (value) => {
			title.textContent = value || `Scenario ${index + 1}`;
		},
	}));
	grid.appendChild(createCheckboxField({
		label: "Active",
		path: `scenarios[${index}].active`,
		value: scenario.active,
		tooltip: "Toggle whether this scenario participates in the simulation run.",
	}));
	card.appendChild(grid);

	card.appendChild(createEventCollection(scenario.events, `scenarios[${index}].events`, {
		heading: "Scenario events",
		titlePrefix: "Event",
		addLabel: "Add event",
	}));

	card.appendChild(createLoanCollection(scenario.loans, `scenarios[${index}].loans`, {
		heading: "Scenario loans",
		addLabel: "Add loan",
	}));

	return card;
}

function createEventCollection(events, basePath, options = {}) {
	const container = document.createElement("div");
	container.className = "editor-subsection";

	if (options.heading) {
		const heading = document.createElement("h4");
		heading.textContent = options.heading;
		if (options.headingClass) {
			heading.classList.add(options.headingClass);
		}
		container.appendChild(heading);
	}

	const collection = document.createElement("div");
	collection.className = "editor-collection";

	if (events.length === 0) {
		const emptyState = document.createElement("p");
		emptyState.className = "muted-text";
		emptyState.textContent = options.emptyMessage || "No events configured.";
		collection.appendChild(emptyState);
	} else {
		events.forEach((event, index) => {
			const card = createEventCard(event, `${basePath}[${index}]`, index, options.titlePrefix || "Event", () => {
				events.splice(index, 1);
				renderConfigEditor();
			});
			collection.appendChild(card);
		});
	}

	container.appendChild(collection);

	const actions = document.createElement("div");
	actions.className = "collection-actions";
	const addButton = document.createElement("button");
	addButton.type = "button";
	addButton.textContent = options.addLabel || "Add event";
	addButton.addEventListener("click", () => {
		events.push(createEmptyEvent());
		renderConfigEditor();
		switchTab("config");
	});
	actions.appendChild(addButton);
	container.appendChild(actions);

	return container;
}

function createEventCard(event, basePath, index, titlePrefix, onRemove) {
	const card = document.createElement("div");
	card.className = "editor-card";

	const { header, title } = createCardHeader(
		event.name || `${titlePrefix} ${index + 1}`,
		onRemove,
		"Remove event",
	);
	card.appendChild(header);

	const grid = document.createElement("div");
	grid.className = "editor-grid";
	grid.appendChild(createInputField({
		label: "Name",
		path: `${basePath}.name`,
		value: event.name ?? "",
		inputType: "text",
		tooltip: "Optional label shown in reports and logs for this event.",
		validation: { type: "text", maxLength: 120 },
		maxLength: 120,
		onChange: (value) => {
			title.textContent = value || `${titlePrefix} ${index + 1}`;
		},
	}));
	grid.appendChild(createInputField({
		label: "Amount",
		path: `${basePath}.amount`,
		value: event.amount ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Positive amounts represent income; negative amounts represent expenses.",
		validation: { type: "number" },
	}));
	grid.appendChild(createInputField({
		label: "Frequency (months)",
		path: `${basePath}.frequency`,
		value: event.frequency ?? "",
		inputType: "number",
		step: "1",
		numberKind: "int",
		tooltip: "Number of months between occurrences (1 = monthly, 3 = quarterly, etc.).",
		validation: { type: "integer", min: 1 },
	}));
	grid.appendChild(createInputField({
		label: "Start date (YYYY-MM)",
		path: `${basePath}.startDate`,
		value: event.startDate ?? "",
		inputType: "month",
		tooltip: "Optional month when this event begins (YYYY-MM).",
		validation: { type: "month" },
		maxLength: 7,
	}));
	grid.appendChild(createInputField({
		label: "End date (YYYY-MM)",
		path: `${basePath}.endDate`,
		value: event.endDate ?? "",
		inputType: "month",
		tooltip: "Optional month when this event ends (YYYY-MM).",
		validation: { type: "month" },
		maxLength: 7,
	}));
	card.appendChild(grid);

	return card;
}

function createLoanCollection(loans, basePath, options = {}) {
	const container = document.createElement("div");
	container.className = "editor-subsection";

	if (options.heading) {
		const heading = document.createElement("h4");
		heading.textContent = options.heading;
		container.appendChild(heading);
	}

	const collection = document.createElement("div");
	collection.className = "editor-collection";

	if (loans.length === 0) {
		const emptyState = document.createElement("p");
		emptyState.className = "muted-text";
		emptyState.textContent = options.emptyMessage || "No loans configured.";
		collection.appendChild(emptyState);
	} else {
		loans.forEach((loan, index) => {
			const card = createLoanCard(loan, `${basePath}[${index}]`, index, () => {
				loans.splice(index, 1);
				renderConfigEditor();
			});
			collection.appendChild(card);
		});
	}

	container.appendChild(collection);

	const actions = document.createElement("div");
	actions.className = "collection-actions";
	const addButton = document.createElement("button");
	addButton.type = "button";
	addButton.textContent = options.addLabel || "Add loan";
	addButton.addEventListener("click", () => {
		loans.push(createEmptyLoan());
		renderConfigEditor();
		switchTab("config");
	});
	actions.appendChild(addButton);
	container.appendChild(actions);

	return container;
}

function createLoanCard(loan, basePath, index, onRemove) {
	const card = document.createElement("div");
	card.className = "editor-card";

	const { header, title } = createCardHeader(
		loan.name || `Loan ${index + 1}`,
		onRemove,
		"Remove loan",
	);
	card.appendChild(header);

	const grid = document.createElement("div");
	grid.className = "editor-grid";
	grid.appendChild(createInputField({
		label: "Name",
		path: `${basePath}.name`,
		value: loan.name ?? "",
		inputType: "text",
		tooltip: "Optional label shown in reports for this loan.",
		validation: { type: "text", maxLength: 120 },
		maxLength: 120,
		onChange: (value) => {
			title.textContent = value || `Loan ${index + 1}`;
		},
	}));
	grid.appendChild(createInputField({
		label: "Principal",
		path: `${basePath}.principal`,
		value: loan.principal ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Original loan principal before any down payment is applied.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Down payment",
		path: `${basePath}.downPayment`,
		value: loan.downPayment ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Amount paid up front to reduce the principal.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Interest rate (%)",
		path: `${basePath}.interestRate`,
		value: loan.interestRate ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Annual interest rate expressed as a percentage.",
		validation: { type: "number", min: 0, max: 100 },
	}));
	grid.appendChild(createInputField({
		label: "Term (months)",
		path: `${basePath}.term`,
		value: loan.term ?? "",
		inputType: "number",
		step: "1",
		numberKind: "int",
		tooltip: "Length of the loan in months.",
		validation: { type: "integer", min: 1 },
	}));
	grid.appendChild(createInputField({
		label: "Start date (YYYY-MM)",
		path: `${basePath}.startDate`,
		value: loan.startDate ?? "",
		inputType: "month",
		tooltip: "Month the loan begins (YYYY-MM).",
		validation: { type: "month" },
		maxLength: 7,
	}));
	grid.appendChild(createInputField({
		label: "Escrow",
		path: `${basePath}.escrow`,
		value: loan.escrow ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Optional monthly escrow payment associated with the loan.",
		validation: { type: "number" },
	}));
	grid.appendChild(createInputField({
		label: "Mortgage insurance",
		path: `${basePath}.mortgageInsurance`,
		value: loan.mortgageInsurance ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Monthly mortgage insurance premium, if applicable.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Mortgage insurance cutoff (%)",
		path: `${basePath}.mortgageInsuranceCutoff`,
		value: loan.mortgageInsuranceCutoff ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Loan-to-value percentage at which mortgage insurance ends.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Early payoff threshold",
		path: `${basePath}.earlyPayoffThreshold`,
		value: loan.earlyPayoffThreshold ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Positive balance buffer that triggers an early payoff.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Early payoff date (YYYY-MM)",
		path: `${basePath}.earlyPayoffDate`,
		value: loan.earlyPayoffDate ?? "",
		inputType: "month",
		tooltip: "Optional month when the loan should be paid off early (YYYY-MM).",
		validation: { type: "month" },
		maxLength: 7,
	}));
	grid.appendChild(createCheckboxField({
		label: "Sell property when paid off",
		path: `${basePath}.sellProperty`,
		value: loan.sellProperty,
		tooltip: "When enabled, the property is sold as soon as the loan is paid off.",
	}));
	grid.appendChild(createInputField({
		label: "Sell price",
		path: `${basePath}.sellPrice`,
		value: loan.sellPrice ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Expected sale price when the property is sold.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Sell costs (net)",
		path: `${basePath}.sellCostsNet`,
		value: loan.sellCostsNet ?? "",
		inputType: "number",
		step: "0.01",
		tooltip: "Net costs (positive) or proceeds (negative) incurred when selling.",
		validation: { type: "number" },
	}));
	card.appendChild(grid);

	const extraPayments = createEventCollection(loan.extraPrincipalPayments, `${basePath}.extraPrincipalPayments`, {
		heading: "Extra principal payments",
		titlePrefix: "Payment",
		addLabel: "Add extra payment",
		emptyMessage: "No extra principal payments configured.",
	});
	card.appendChild(extraPayments);

	return card;
}

function createCardHeader(titleText, onRemove, removeLabel, options = {}) {
	const header = document.createElement("div");
	header.className = "editor-card-header";
	if (options.extraClass) {
		header.classList.add(options.extraClass);
	}

	const title = document.createElement("h4");
	title.textContent = titleText;
	header.appendChild(title);

	if (typeof onRemove === "function") {
		const actions = document.createElement("div");
		actions.className = "editor-inline-actions";
		const removeButton = document.createElement("button");
		removeButton.type = "button";
		removeButton.textContent = removeLabel || "Remove";
		removeButton.addEventListener("click", onRemove);
		actions.appendChild(removeButton);
		header.appendChild(actions);
	}

	return { header, title };
}

function createInputField({
	label,
	path,
	value,
	inputType = "text",
	placeholder = "",
	step,
	min,
	options = [],
	numberKind,
	onChange,
	tooltip = "",
	validation = null,
	maxLength,
}) {
	const wrapper = document.createElement("label");
	wrapper.className = "editor-field";
	const labelEl = document.createElement("span");
	labelEl.className = "editor-label";
	labelEl.textContent = label;
	wrapper.appendChild(labelEl);

	let control;

	if (inputType === "select") {
		control = document.createElement("select");
		options.forEach((option) => {
			const opt = document.createElement("option");
			opt.value = option.value;
			opt.textContent = option.label;
			if (option.value === (value ?? "")) {
				opt.selected = true;
			}
			control.appendChild(opt);
		});
		control.dataset.valueType = "text";
	} else {
		control = document.createElement("input");
		control.type = inputType;
		control.value = value ?? "";
		control.placeholder = placeholder;

		if (inputType === "number") {
			control.step = step ?? "any";
			if (min !== undefined) {
				control.min = String(min);
			}
			control.dataset.valueType = "number";
			if (numberKind) {
				control.dataset.numberKind = numberKind;
			}
			control.inputMode = numberKind === "int" ? "numeric" : "decimal";
		} else {
			control.dataset.valueType = "text";
			if (inputType === "month") {
				control.inputMode = "numeric";
			}
		}
	}

	if (typeof maxLength === "number" && control.tagName === "INPUT") {
		control.maxLength = maxLength;
	}

	if (tooltip) {
		control.title = tooltip;
		wrapper.title = tooltip;
	}

	if (validation && validation.type === "month") {
		control.setAttribute("pattern", "\\d{4}-(0[1-9]|1[0-2])");
	}

	control.dataset.path = path;

	const errorEl = document.createElement("span");
	errorEl.className = "field-error hidden";

	const entry = {
		control,
		errorEl,
		validation,
		label,
		path,
		touched: false,
		isValid: !validation,
	};

	const eventType = inputType === "select" ? "change" : "input";
	control.addEventListener(eventType, (event) => {
		updateFromInput(event, onChange);
		entry.touched = true;
		if (entry.validation) {
			runFieldValidation(entry);
		} else {
			entry.isValid = true;
		}
		updateEditorActionsState();
	});

	control.addEventListener("blur", () => {
		if (!entry.validation) {
			return;
		}
		entry.touched = true;
		runFieldValidation(entry, { report: true });
		updateEditorActionsState();
	});

	registeredInputs.push(entry);
	if (entry.validation) {
		runFieldValidation(entry, { report: false });
	} else {
		entry.isValid = true;
	}

	wrapper.appendChild(control);
	wrapper.appendChild(errorEl);
	return wrapper;
}

function createCheckboxField({ label, path, value, tooltip = "" }) {
	const wrapper = document.createElement("label");
	wrapper.className = "editor-field checkbox-field";

	const input = document.createElement("input");
	input.type = "checkbox";
	input.checked = Boolean(value);
	input.dataset.path = path;
	input.dataset.valueType = "boolean";
	input.addEventListener("change", (event) => updateFromInput(event));

	if (tooltip) {
		input.title = tooltip;
		wrapper.title = tooltip;
	}

	const labelEl = document.createElement("span");
	labelEl.className = "editor-label";
	labelEl.textContent = label;

	wrapper.appendChild(input);
	wrapper.appendChild(labelEl);
	return wrapper;
}

function validateEditorForm({ focusFirstError = false, report = false } = {}) {
	let firstInvalid = null;
	registeredInputs.forEach((entry) => {
		if (!runFieldValidation(entry, { report })) {
			if (!firstInvalid) {
				firstInvalid = entry;
			}
		}
	});

	if (firstInvalid && focusFirstError) {
		try {
			firstInvalid.control.focus({ preventScroll: true });
		} catch (error) {
			firstInvalid.control.focus();
		}
		if (typeof firstInvalid.control.scrollIntoView === "function") {
			firstInvalid.control.scrollIntoView({ block: "center", behavior: "smooth" });
		}
	}

	updateEditorActionsState();
	return !firstInvalid;
}

function runFieldValidation(entry, options = {}) {
	const { validation } = entry;
	if (!validation) {
		entry.isValid = true;
		return true;
	}

	const { control, errorEl } = entry;
	const rawValue = control.value ?? "";
	const value = typeof rawValue === "string" ? rawValue.trim() : rawValue;
	const required = Boolean(validation.required);
	const hasValue = value !== "";
	let message = "";

	if (required && !hasValue) {
		message = validation.requiredMessage || `${entry.label || "This field"} is required.`;
	}

	let numericValue;
	if (!message && hasValue) {
		switch (validation.type) {
			case "month":
				if (!MONTH_PATTERN.test(value)) {
					message = validation.formatMessage || "Enter a valid month in YYYY-MM.";
				}
				break;
			case "integer":
				numericValue = Number(value);
				if (!Number.isInteger(numericValue)) {
					message = validation.formatMessage || `${entry.label || "This field"} must be a whole number.`;
				}
				break;
			case "number":
				numericValue = Number(value);
				if (Number.isNaN(numericValue)) {
					message = validation.formatMessage || `${entry.label || "This field"} must be a number.`;
				}
				break;
			case "text":
				if (validation.pattern && !validation.pattern.test(value)) {
					message = validation.formatMessage || `${entry.label || "This field"} has an invalid format.`;
				}
				if (!message && validation.maxLength && value.length > validation.maxLength) {
					message = validation.maxLengthMessage || `${entry.label || "This field"} must be ${validation.maxLength} characters or fewer.`;
				}
				break;
			default:
				break;
		}
	}

	if (!message && hasValue && (validation.type === "number" || validation.type === "integer")) {
		if (numericValue === undefined) {
			numericValue = Number(value);
		}
		if (!Number.isNaN(numericValue)) {
			if (validation.min !== undefined && numericValue < validation.min) {
				message = validation.minMessage || `${entry.label || "This field"} must be at least ${validation.min}.`;
			} else if (validation.max !== undefined && numericValue > validation.max) {
				message = validation.maxMessage || `${entry.label || "This field"} must be at most ${validation.max}.`;
			}
		}
	}

	if (!message && typeof validation.validate === "function") {
		const customMessage = validation.validate(value, control);
		if (typeof customMessage === "string" && customMessage) {
			message = customMessage;
		}
	}

	entry.isValid = !message;
	const shouldReveal = options.report || entry.touched;

	if (validation.required) {
		control.setAttribute("aria-required", "true");
	} else {
		control.removeAttribute("aria-required");
	}

	if (shouldReveal) {
		if (message) {
			errorEl.textContent = message;
			errorEl.classList.remove("hidden");
			control.classList.add("invalid");
			control.setAttribute("aria-invalid", "true");
		} else {
			errorEl.textContent = "";
			errorEl.classList.add("hidden");
			control.classList.remove("invalid");
			control.removeAttribute("aria-invalid");
		}
	} else {
		errorEl.textContent = "";
		errorEl.classList.add("hidden");
		control.classList.remove("invalid");
		control.removeAttribute("aria-invalid");
	}

	return entry.isValid;
}

function createEmptyEvent() {
	return {
		name: "",
		amount: 0,
		frequency: 1,
	};
}

function createEmptyLoan() {
	return {
		name: "",
		principal: 0,
		downPayment: 0,
		interestRate: 0,
		term: 0,
		startDate: "",
		escrow: 0,
		mortgageInsurance: 0,
		mortgageInsuranceCutoff: 0,
		earlyPayoffThreshold: 0,
		earlyPayoffDate: "",
		sellProperty: false,
		sellPrice: 0,
		sellCostsNet: 0,
		extraPrincipalPayments: [],
	};
}

function createEmptyScenario() {
	return {
		name: "",
		active: true,
		events: [],
		loans: [],
	};
}

function createInitialConfig() {
	return {
		startDate: "",
		output: { format: "pretty" },
		common: {
			startingValue: "",
			deathDate: "",
			events: [],
			loans: [],
		},
		scenarios: [createEmptyScenario()],
	};
}

function getDefaultLoggingConfig() {
	return {
		level: "info",
		format: "json",
	};
}

function setDataAvailability(available) {
	dataAvailable = available;
	if (resultsTabButton) {
		resultsTabButton.disabled = !available;
	}
	if (!available && activeTab === "results") {
		switchTab("config");
	}

	updateStickyMetrics();
	updateEditorActionsState();
}

function updateEditorActionsState() {
	const hasConfig = !!currentConfig;
	const hasErrors = hasValidationErrors();
	runForecastButton.disabled = isEditorLoading || !hasConfig || hasErrors;
	downloadConfigButton.disabled = isEditorLoading || !hasConfig;
}

function hasValidationErrors() {
	return registeredInputs.some((entry) => entry.validation && entry.isValid === false);
}

function switchTab(tabName) {
	const targetPanel = tabPanels[tabName];
	if (!targetPanel) {
		return;
	}

	const targetButton = tabButtons.find((button) => button.dataset.tab === tabName);
	if (targetButton && targetButton.disabled) {
		return;
	}

	activeTab = tabName;

	tabButtons.forEach((button) => {
		const isActive = button.dataset.tab === tabName;
		button.classList.toggle("active", isActive);
		button.setAttribute("aria-selected", isActive ? "true" : "false");
	});

	Object.entries(tabPanels).forEach(([name, panel]) => {
		const isActive = name === tabName;
		panel.classList.toggle("active", isActive);
		panel.hidden = !isActive;
	});

	updateStickyMetrics();
}

async function handleRunForecast() {
	if (!currentConfig) {
		return;
	}

	const isValid = validateEditorForm({ focusFirstError: true, report: true });
	if (!isValid) {
		showMessage("Please fix the highlighted fields before running the forecast.", "error");
		return;
	}

	toggleEditorLoading(true);
	showMessage("", null);

	try {
		const payload = buildConfigPayload();
		const response = await fetch("/api/editor/forecast", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(payload),
		});

		const data = await response.json();

		if (!response.ok) {
			throw new Error(data.error || "Unable to process forecast");
		}

		processForecastResponse(data, "Forecast updated successfully.");
	} catch (error) {
		console.error("Run request failed", error);
		showMessage(error.message, "error");
	} finally {
		toggleEditorLoading(false);
	}
}

function handleResetConfig() {
	const confirmed = window.confirm("Reset the current configuration? This will clear all fields and results.");
	if (!confirmed) {
		return;
	}

	currentConfig = createInitialConfig();
	hiddenLogging = getDefaultLoggingConfig();
	latestConfigYaml = "";
	clearResultsView();
	if (configDownloadUrl) {
		URL.revokeObjectURL(configDownloadUrl);
		configDownloadUrl = null;
	}
	setDataAvailability(false);
	renderConfigEditor();
	switchTab("config");
	showMessage("Configuration reset. Start building your new plan.", "success");
}

function initializeWorkspace() {
	if (!defaultConfigInitialized) {
		currentConfig = createInitialConfig();
		hiddenLogging = getDefaultLoggingConfig();
		defaultConfigInitialized = true;
	}

	clearResultsView();
	renderConfigEditor();
	setDataAvailability(false);
	switchTab("config");
	showMessage("Start by building a plan or upload an existing YAML configuration.", null);
}

function buildConfigPayload(options = {}) {
	const { includeDefaults = false } = options;
	const payload = cloneDeep(currentConfig) || {};

	if (!payload.output || typeof payload.output !== "object") {
		payload.output = {};
	}
	if (includeDefaults && !payload.output.format) {
		payload.output.format = "pretty";
	}

	if (hiddenLogging) {
		payload.logging = cloneDeep(hiddenLogging);
	} else if (includeDefaults) {
		payload.logging = getDefaultLoggingConfig();
	}

	return payload;
}

async function downloadCurrentConfig() {
	if (!currentConfig) {
		return;
	}

	downloadConfigButton.disabled = true;
	showMessage("", null);

	try {
		const payload = buildConfigPayload({ includeDefaults: true });
		const response = await fetch("/api/editor/export", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(payload),
		});

		const data = await response.json();

		if (!response.ok) {
			throw new Error(data.error || "Unable to prepare configuration download");
		}

		latestConfigYaml = data.configYaml || "";
		triggerConfigDownload(latestConfigYaml);
		showMessage("Configuration downloaded.", "success");
	} catch (error) {
		console.error("Download config failed", error);
		showMessage(error.message, "error");
	} finally {
		const hasConfig = !!currentConfig;
		downloadConfigButton.disabled = !hasConfig;
		runForecastButton.disabled = !hasConfig;
	}
}

function triggerConfigDownload(yamlContent) {
	if (!yamlContent) {
		return;
	}

	if (configDownloadUrl) {
		URL.revokeObjectURL(configDownloadUrl);
	}

	const blob = new Blob([yamlContent], { type: "text/yaml" });
	configDownloadUrl = URL.createObjectURL(blob);

	const anchor = document.createElement("a");
	anchor.href = configDownloadUrl;
	anchor.download = `config-${new Date().toISOString().split("T")[0]}.yaml`;
	document.body.appendChild(anchor);
	anchor.click();
	document.body.removeChild(anchor);
}

function cloneDeep(value) {
	if (value === null || value === undefined) {
		return value;
	}
	if (typeof structuredClone === "function") {
		try {
			return structuredClone(value);
		} catch (error) {
			console.error("structuredClone failed, falling back to JSON method in cloneDeep:", error);
		}
	}
	return JSON.parse(JSON.stringify(value));
}

function updateFromInput(event, onChange) {
	const target = event.target;
	const path = target.dataset.path;
	if (!path) {
		return;
	}

	const valueType = target.dataset.valueType || "text";
	let value;

	if (valueType === "boolean") {
		value = target.checked;
	} else if (valueType === "number") {
		if (target.value === "") {
			value = null;
		} else {
			const kind = target.dataset.numberKind === "int" ? "int" : "float";
			const parsed = kind === "int" ? parseInt(target.value, 10) : parseFloat(target.value);
			if (Number.isNaN(parsed)) {
				return;
			}
			value = parsed;
		}
	} else {
		value = target.value.trim();
	}

	updateConfigAtPath(path, value, valueType, target.dataset.numberKind);

	if (typeof onChange === "function") {
		onChange(typeof value === "string" ? value : target.value);
	}
}

function updateConfigAtPath(path, value, valueType, numberKind) {
	if (!currentConfig) {
		return;
	}

	const segments = splitPath(path);
	if (segments.length === 0) {
		return;
	}

	const { container, key } = getContainerByPath(currentConfig, segments, true);
	if (!container || key === undefined) {
		return;
	}

	if (valueType === "number") {
		if (value === null) {
			delete container[key];
		} else if (numberKind === "int") {
			container[key] = Math.trunc(value);
		} else {
			container[key] = value;
		}
	} else if (valueType === "boolean") {
		container[key] = Boolean(value);
	} else {
		if (!value) {
			delete container[key];
		} else {
			container[key] = value;
		}
	}
}

function splitPath(path) {
	const result = [];
	const pattern = /[^.\[\]]+|\[(\d+)\]/g;
	let match;
	while ((match = pattern.exec(path)) !== null) {
		if (match[1] !== undefined) {
			result.push(Number(match[1]));
		} else {
			result.push(match[0]);
		}
	}
	return result;
}

function getContainerByPath(root, segments, createMissing) {
	let current = root;
	for (let i = 0; i < segments.length - 1; i += 1) {
		const segment = segments[i];
		const nextSegment = segments[i + 1];

		if (typeof segment === "number") {
			if (!Array.isArray(current)) {
				return {};
			}
			if (!current[segment]) {
				if (createMissing) {
					current[segment] = typeof nextSegment === "number" ? [] : {};
				} else {
					return {};
				}
			}
			current = current[segment];
		} else {
			if (!(segment in current) || current[segment] == null) {
				if (createMissing) {
					current[segment] = typeof nextSegment === "number" ? [] : {};
				} else {
					return {};
				}
			}
			current = current[segment];
		}
	}

	return { container: current, key: segments[segments.length - 1] };
}
