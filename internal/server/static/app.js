const form = document.getElementById("upload-form");
const fileInput = document.getElementById("config-file");
const runButton = document.getElementById("run-button");
const loadingIndicator = document.getElementById("loading-indicator");
const messageEl = document.getElementById("message");
const warningsEl = document.getElementById("warnings");
const resultsPanel = document.getElementById("results-section");
const configPanel = document.getElementById("config-panel");
const uploadPanel = document.getElementById("upload-panel");
const tableHead = document.querySelector("#results-table thead");
const tableBody = document.querySelector("#results-table tbody");
const downloadLink = document.getElementById("download-link");
const durationEl = document.getElementById("duration");
const configEditorRoot = document.getElementById("config-editor");
const rerunButton = document.getElementById("rerun-button");
const downloadConfigButton = document.getElementById("download-config-button");
const editorLoading = document.getElementById("editor-loading");
const editorMessage = document.getElementById("editor-message");
const tablistContainer = document.querySelector(".tablist-container");
if (configPanel) {
	configPanel.classList.add("sticky-headers");
}

const rootStyle = document.documentElement.style;

const tabButtons = Array.from(document.querySelectorAll(".tab-button"));
const tabPanels = {
	upload: uploadPanel,
	results: resultsPanel,
	config: configPanel,
};

let activeTab = "upload";
let dataAvailable = false;
let currentObjectUrl = null;
let configDownloadUrl = null;
let currentConfig = null;
let hiddenLogging = null;
let latestConfigYaml = "";

form.addEventListener("submit", handleUpload);
tabButtons.forEach((button) => {
	button.addEventListener("click", () => {
		const tab = button.dataset.tab;
		if (tab) {
			switchTab(tab);
		}
	});
});

rerunButton.addEventListener("click", handleRerun);
downloadConfigButton.addEventListener("click", downloadCurrentConfig);

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

function toggleLoading(isLoading) {
	runButton.disabled = isLoading;
	loadingIndicator.classList.toggle("hidden", !isLoading);
}

function toggleEditorLoading(isLoading) {
	const configReady = dataAvailable && !!currentConfig;
	rerunButton.disabled = isLoading || !configReady;
	downloadConfigButton.disabled = isLoading || !configReady;
	editorLoading.classList.toggle("hidden", !isLoading);
	updateStickyMetrics();
}

async function handleUpload(event) {
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

		processForecastResponse(data, "Forecast completed successfully.");
	} catch (error) {
		console.error("Forecast request failed", error);
		showMessage(error.message, "error");
	} finally {
		toggleLoading(false);
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
	messageEl.textContent = message;
	messageEl.className = type ? `message ${type}` : "message";
}

function clearResults() {
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

	if (configDownloadUrl) {
		URL.revokeObjectURL(configDownloadUrl);
		configDownloadUrl = null;
	}

	currentConfig = null;
	hiddenLogging = null;
	latestConfigYaml = "";
	configEditorRoot.innerHTML = "";
	showEditorMessage("", null);
	setDataAvailability(false);
}

function renderResults(data) {
	if (!data || !Array.isArray(data.scenarios) || !Array.isArray(data.rows)) {
		throw new Error("Malformed response received from server");
	}

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
		currentConfig = null;
		hiddenLogging = null;
		latestConfigYaml = "";
		renderConfigEditor();
		setDataAvailability(false);
		return;
	}

	const prepared = prepareConfigForEditing(rawConfig);
	currentConfig = prepared.config;
	hiddenLogging = prepared.logging;
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

	cloned.output = cloned.output && typeof cloned.output === "object" ? cloned.output : {};

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

	if (!currentConfig) {
		const emptyState = document.createElement("p");
		emptyState.className = "muted-text";
		emptyState.textContent = "Upload a configuration to view and edit its settings.";
		configEditorRoot.appendChild(emptyState);
		updateStickyMetrics();
		return;
	}

	showEditorMessage("", null);

	const simulationSection = createSection("Simulation", "Control global simulation behavior.");
	const simGrid = document.createElement("div");
	simGrid.className = "editor-grid";
	simGrid.appendChild(createInputField({
		label: "Start date (YYYY-MM)",
		path: "startDate",
		value: currentConfig.startDate ?? "",
		inputType: "month",
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
	}));
	commonGrid.appendChild(createInputField({
		label: "Death date (YYYY-MM)",
		path: "common.deathDate",
		value: currentConfig.common.deathDate ?? "",
		inputType: "month",
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
		onChange: (value) => {
			title.textContent = value || `Scenario ${index + 1}`;
		},
	}));
	grid.appendChild(createCheckboxField({
		label: "Active",
		path: `scenarios[${index}].active`,
		value: scenario.active,
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
	}));
	grid.appendChild(createInputField({
		label: "Frequency (months)",
		path: `${basePath}.frequency`,
		value: event.frequency ?? "",
		inputType: "number",
		step: "1",
		numberKind: "int",
	}));
	grid.appendChild(createInputField({
		label: "Start date (YYYY-MM)",
		path: `${basePath}.startDate`,
		value: event.startDate ?? "",
		inputType: "month",
	}));
	grid.appendChild(createInputField({
		label: "End date (YYYY-MM)",
		path: `${basePath}.endDate`,
		value: event.endDate ?? "",
		inputType: "month",
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
	}));
	grid.appendChild(createInputField({
		label: "Down payment",
		path: `${basePath}.downPayment`,
		value: loan.downPayment ?? "",
		inputType: "number",
		step: "0.01",
	}));
	grid.appendChild(createInputField({
		label: "Interest rate (%)",
		path: `${basePath}.interestRate`,
		value: loan.interestRate ?? "",
		inputType: "number",
		step: "0.01",
	}));
	grid.appendChild(createInputField({
		label: "Term (months)",
		path: `${basePath}.term`,
		value: loan.term ?? "",
		inputType: "number",
		step: "1",
		numberKind: "int",
	}));
	grid.appendChild(createInputField({
		label: "Start date (YYYY-MM)",
		path: `${basePath}.startDate`,
		value: loan.startDate ?? "",
		inputType: "month",
	}));
	grid.appendChild(createInputField({
		label: "Escrow",
		path: `${basePath}.escrow`,
		value: loan.escrow ?? "",
		inputType: "number",
		step: "0.01",
	}));
	grid.appendChild(createInputField({
		label: "Mortgage insurance",
		path: `${basePath}.mortgageInsurance`,
		value: loan.mortgageInsurance ?? "",
		inputType: "number",
		step: "0.01",
	}));
	grid.appendChild(createInputField({
		label: "Mortgage insurance cutoff (%)",
		path: `${basePath}.mortgageInsuranceCutoff`,
		value: loan.mortgageInsuranceCutoff ?? "",
		inputType: "number",
		step: "0.01",
	}));
	grid.appendChild(createInputField({
		label: "Early payoff threshold",
		path: `${basePath}.earlyPayoffThreshold`,
		value: loan.earlyPayoffThreshold ?? "",
		inputType: "number",
		step: "0.01",
	}));
	grid.appendChild(createInputField({
		label: "Early payoff date (YYYY-MM)",
		path: `${basePath}.earlyPayoffDate`,
		value: loan.earlyPayoffDate ?? "",
		inputType: "month",
	}));
	grid.appendChild(createCheckboxField({
		label: "Sell property when paid off",
		path: `${basePath}.sellProperty`,
		value: loan.sellProperty,
	}));
	grid.appendChild(createInputField({
		label: "Sell price",
		path: `${basePath}.sellPrice`,
		value: loan.sellPrice ?? "",
		inputType: "number",
		step: "0.01",
	}));
	grid.appendChild(createInputField({
		label: "Sell costs (net)",
		path: `${basePath}.sellCostsNet`,
		value: loan.sellCostsNet ?? "",
		inputType: "number",
		step: "0.01",
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

function createInputField({ label, path, value, inputType = "text", placeholder = "", step, min, options = [], numberKind, onChange }) {
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
		} else {
			control.dataset.valueType = "text";
		}
	}

	control.dataset.path = path;

	const eventType = inputType === "select" ? "change" : "input";
	control.addEventListener(eventType, (event) => updateFromInput(event, onChange));

	wrapper.appendChild(control);
	return wrapper;
}

function createCheckboxField({ label, path, value }) {
	const wrapper = document.createElement("label");
	wrapper.className = "editor-field checkbox-field";

	const input = document.createElement("input");
	input.type = "checkbox";
	input.checked = Boolean(value);
	input.dataset.path = path;
	input.dataset.valueType = "boolean";
	input.addEventListener("change", (event) => updateFromInput(event));

	const labelEl = document.createElement("span");
	labelEl.className = "editor-label";
	labelEl.textContent = label;

	wrapper.appendChild(input);
	wrapper.appendChild(labelEl);
	return wrapper;
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

function setDataAvailability(available) {
	dataAvailable = available;
	const resultsTab = document.getElementById("tab-results");
	const configTab = document.getElementById("tab-config");
	const configReady = available && !!currentConfig;

	resultsTab.disabled = !available;
	configTab.disabled = !available;

	if (!available && activeTab !== "upload") {
		switchTab("upload");
	}

	rerunButton.disabled = !configReady;
	downloadConfigButton.disabled = !configReady;

	updateStickyMetrics();
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

function showEditorMessage(message, type) {
	if (!message) {
		editorMessage.className = "message hidden";
		updateStickyMetrics();
		return;
	}

	editorMessage.textContent = message;
	editorMessage.className = type ? `message ${type}` : "message";
	editorMessage.classList.remove("hidden");
	updateStickyMetrics();
}

async function handleRerun() {
	if (!currentConfig) {
		return;
	}

	toggleEditorLoading(true);
	showEditorMessage("", null);

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

		processForecastResponse(data, "Forecast updated successfully.", { switchToResults: false });
		showEditorMessage("Forecast updated successfully.", "success");
	} catch (error) {
		console.error("Re-run request failed", error);
		showEditorMessage(error.message, "error");
		showMessage(error.message, "error");
	} finally {
		toggleEditorLoading(false);
	}
}

function buildConfigPayload() {
	const payload = cloneDeep(currentConfig) || {};
	if (hiddenLogging) {
		payload.logging = cloneDeep(hiddenLogging);
	}
	return payload;
}

async function downloadCurrentConfig() {
	if (!currentConfig) {
		return;
	}

	downloadConfigButton.disabled = true;
	showEditorMessage("", null);

	try {
		const payload = buildConfigPayload();
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
		showEditorMessage("Configuration downloaded.", "success");
	} catch (error) {
		console.error("Download config failed", error);
		showEditorMessage(error.message, "error");
		showMessage(error.message, "error");
	} finally {
		const configReady = dataAvailable && !!currentConfig;
		downloadConfigButton.disabled = !configReady;
		rerunButton.disabled = !configReady;
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
			// Fallback to JSON method below
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
