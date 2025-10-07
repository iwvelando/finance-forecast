const messageEl = document.getElementById("message");
const warningsEl = document.getElementById("warnings");
const resultsPanel = document.getElementById("results-section");
const configPanel = document.getElementById("config-panel");
const tableHead = document.querySelector("#results-table thead");
const tableBody = document.querySelector("#results-table tbody");
const downloadLink = document.getElementById("download-link");
const durationEl = document.getElementById("duration");
const scenarioTabsEl = document.getElementById("scenario-tabs");
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
const rootElement = document.documentElement;

const tabButtons = Array.from(document.querySelectorAll(".tab-button"));
const tabPanels = {
	results: resultsPanel,
	config: configPanel,
};
const resultsTabButton = document.getElementById("tab-results");
const themeToggleButtons = Array.from(document.querySelectorAll(".theme-toggle"));

const ARROW_STEP_LARGE = 100;
const ARROW_STEP_SMALL = 1;

let activeTab = "config";
let dataAvailable = false;
let currentObjectUrl = null;
let configDownloadUrl = null;
let currentConfig = null;
let hiddenLogging = null;
let latestConfigYaml = "";
let latestCsvContent = "";
let latestCsvFilename = "";
let defaultConfigInitialized = false;
let isEditorLoading = false;
let registeredInputs = [];
let tooltipCounter = 0;
let activeHelpTooltip = null;
let helpTooltipInitialized = false;
let forecastDataset = null;
let activeScenarioIndex = 0;
let latestForecastResponse = null;
const THEME_STORAGE_KEY = "financeForecast.theme";

const MONTH_PATTERN = /^\d{4}-(0[1-9]|1[0-2])$/;

function getCurrentMonthValue() {
	const now = new Date();
	const year = now.getFullYear();
	const month = String(now.getMonth() + 1).padStart(2, "0");
	return `${year}-${month}`;
}

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
if (downloadLink) {
	downloadLink.addEventListener("click", handleCsvDownloadClick);
}

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
initializeThemeControls();

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

function isSavePickerAvailable() {
	return typeof window !== "undefined" && typeof window.showSaveFilePicker === "function";
}

async function saveBlobWithPickerOrFallback(blob, options = {}) {
	const {
		suggestedName,
		mimeType = "application/octet-stream",
		extensions = [],
		description = "File",
		fallbackDownload,
	} = options;

	if (isSavePickerAvailable()) {
		try {
			const pickerOptions = {
				suggestedName,
				types: [
					{
						description,
						accept: {
							[mimeType]: Array.isArray(extensions) && extensions.length > 0 ? extensions : [`.${(suggestedName || "file").split(".").pop()}`],
						},
					},
				],
			};
			const fileHandle = await window.showSaveFilePicker(pickerOptions);
			const writable = await fileHandle.createWritable();
			await writable.write(blob);
			await writable.close();
			return "saved";
		} catch (error) {
			if (error && error.name === "AbortError") {
				return "cancelled";
			}
			console.warn("Save picker unavailable, falling back to anchor download.", error);
		}
	}

	if (typeof fallbackDownload === "function") {
		try {
			await fallbackDownload();
			return "fallback";
		} catch (fallbackError) {
			console.error("Fallback download failed", fallbackError);
			return "error";
		}
	}

	return "unavailable";
}

function triggerAnchorDownload(url, filename) {
	const anchor = document.createElement("a");
	anchor.href = url;
	anchor.download = filename;
	anchor.rel = "noopener";
	document.body.appendChild(anchor);
	anchor.click();
	document.body.removeChild(anchor);
}

function initializeThemeControls() {
	applyStoredThemePreference();
	setupSystemThemeWatcher();
	if (themeToggleButtons.length === 0) {
		return;
	}

	themeToggleButtons.forEach((button) => {
		button.addEventListener("click", () => {
			const mode = button.dataset.themeMode;
			if (!mode) {
				return;
			}
			setThemePreference(mode);
		});
	});

	updateThemeToggleState(getThemePreference());
}

function getThemePreference() {
	return localStorage.getItem(THEME_STORAGE_KEY) || "system";
}

function setThemePreference(mode) {
	const normalized = mode === "light" || mode === "dark" ? mode : "system";
	localStorage.setItem(THEME_STORAGE_KEY, normalized);
	applyTheme(normalized);
	updateThemeToggleState(normalized);
}

function applyStoredThemePreference() {
	const preference = getThemePreference();
	applyTheme(preference);
}

function applyTheme(mode) {
	rootElement.setAttribute("data-theme", mode);
	if (mode === "dark") {
		rootElement.classList.add("theme-dark");
		rootElement.classList.remove("theme-light");
		rootStyle.setProperty("color-scheme", "dark");
	} else if (mode === "light") {
		rootElement.classList.add("theme-light");
		rootElement.classList.remove("theme-dark");
		rootStyle.setProperty("color-scheme", "light");
	} else {
		rootElement.classList.remove("theme-light", "theme-dark");
		const prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
		rootElement.classList.toggle("theme-dark", prefersDark);
		rootElement.classList.toggle("theme-light", !prefersDark);
		rootStyle.setProperty("color-scheme", prefersDark ? "dark" : "light");
	}
}

function updateThemeToggleState(activeMode) {
	if (themeToggleButtons.length === 0) {
		return;
	}

	themeToggleButtons.forEach((button) => {
		const mode = button.dataset.themeMode;
		if (!mode) {
			return;
		}
		const isActive = mode === activeMode;
		button.classList.toggle("active", isActive);
		button.setAttribute("aria-pressed", isActive ? "true" : "false");
	});
}

function setupSystemThemeWatcher() {
	if (!window.matchMedia) {
		return;
	}
	const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
	const handler = () => {
		if (getThemePreference() !== "system") {
			return;
		}
		applyTheme("system");
		updateThemeToggleState("system");
	};
	if (typeof mediaQuery.addEventListener === "function") {
		mediaQuery.addEventListener("change", handler);
	} else if (typeof mediaQuery.addListener === "function") {
		mediaQuery.addListener(handler);
	}
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
	const scenarios = Array.isArray(data?.scenarios) ? [...data.scenarios] : [];
	const rows = Array.isArray(data?.rows) ? data.rows : [];
	forecastDataset = { scenarios, rows };
	if (scenarios.length === 0) {
		activeScenarioIndex = 0;
	} else if (activeScenarioIndex >= scenarios.length) {
		activeScenarioIndex = scenarios.length - 1;
	}
	latestForecastResponse = data;
	renderResults();
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
	if (scenarioTabsEl) {
		scenarioTabsEl.innerHTML = "";
		scenarioTabsEl.classList.add("hidden");
	}
	forecastDataset = null;
	latestForecastResponse = null;
	activeScenarioIndex = 0;
	latestCsvContent = "";
	latestCsvFilename = "";

	if (currentObjectUrl) {
		URL.revokeObjectURL(currentObjectUrl);
		currentObjectUrl = null;
	}
}

function renderResults() {
	if (!latestForecastResponse || !forecastDataset) {
		throw new Error("No forecast data available to render");
	}

	renderWarnings(latestForecastResponse.warnings);
	renderActiveScenario();
	prepareDownload(latestForecastResponse.csv);

	if (latestForecastResponse.duration) {
		durationEl.textContent = `Computed in ${latestForecastResponse.duration}`;
	} else {
		durationEl.textContent = "";
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

function renderActiveScenario() {
	renderScenarioTabs();
	renderScenarioTable();
}

function clampActiveScenarioIndex() {
	if (!forecastDataset || !Array.isArray(forecastDataset.scenarios) || forecastDataset.scenarios.length === 0) {
		activeScenarioIndex = 0;
		return activeScenarioIndex;
	}

	if (activeScenarioIndex < 0) {
		activeScenarioIndex = 0;
	} else if (activeScenarioIndex >= forecastDataset.scenarios.length) {
		activeScenarioIndex = forecastDataset.scenarios.length - 1;
	}

	return activeScenarioIndex;
}

function renderScenarioTabs() {
	if (!scenarioTabsEl) {
		return;
	}

	scenarioTabsEl.innerHTML = "";

	if (!forecastDataset || !Array.isArray(forecastDataset.scenarios) || forecastDataset.scenarios.length <= 1) {
		scenarioTabsEl.classList.add("hidden");
		return;
	}

	const currentIndex = clampActiveScenarioIndex();

	forecastDataset.scenarios.forEach((rawName, index) => {
		const name = rawName || `Scenario ${index + 1}`;
		const button = document.createElement("button");
		button.type = "button";
		button.className = "scenario-tab";
		if (index === currentIndex) {
			button.classList.add("active");
		}
		button.setAttribute("role", "tab");
		button.setAttribute("aria-selected", index === currentIndex ? "true" : "false");
		button.setAttribute("aria-controls", "results-table");
		button.setAttribute("tabindex", index === currentIndex ? "0" : "-1");
		button.textContent = name;
		button.addEventListener("click", () => {
			if (activeScenarioIndex !== index) {
				activeScenarioIndex = index;
				renderActiveScenario();
				updateStickyMetrics();
			}
		});
		scenarioTabsEl.appendChild(button);
	});

	scenarioTabsEl.classList.remove("hidden");
}

function renderScenarioTable() {
	tableHead.innerHTML = "";
	tableBody.innerHTML = "";

	if (!forecastDataset || !Array.isArray(forecastDataset.scenarios) || forecastDataset.scenarios.length === 0) {
		return;
	}

	const scenarioIndex = clampActiveScenarioIndex();
	const scenarioName = forecastDataset.scenarios[scenarioIndex] || `Scenario ${scenarioIndex + 1}`;

	const headRow = document.createElement("tr");
	headRow.classList.add("primary-header-row");
	headRow.appendChild(createHeaderCell("Date"));

	const scenarioHeader = createHeaderCell(scenarioName);
	scenarioHeader.colSpan = 3;
	scenarioHeader.classList.add("scenario-heading");
	headRow.appendChild(scenarioHeader);
	tableHead.appendChild(headRow);

	const subHeadRow = document.createElement("tr");
	subHeadRow.classList.add("secondary-header-row");
	subHeadRow.appendChild(createHeaderCell("", "subhead"));
	subHeadRow.appendChild(createHeaderCell("Liquid Net Worth", "subhead"));
	subHeadRow.appendChild(createHeaderCell("Total Net Worth", "subhead"));
	subHeadRow.appendChild(createHeaderCell("Notes", "subhead"));
	tableHead.appendChild(subHeadRow);

	const currencyFormatter = new Intl.NumberFormat(undefined, {
		style: "currency",
		currency: "USD",
		minimumFractionDigits: 2,
	});

	const noValueMarkup = '<span class="muted-text">—</span>';
	const rows = Array.isArray(forecastDataset.rows) ? forecastDataset.rows : [];
	rows.forEach((row) => {
		const tr = document.createElement("tr");
		tr.appendChild(createCell(row.date));

		const value = Array.isArray(row.values) ? row.values[scenarioIndex] || {} : {};
		const liquidAmount = typeof value.liquid === "number"
			? value.liquid
			: typeof value.amount === "number"
				? value.amount
				: null;
		const totalAmount = typeof value.total === "number"
			? value.total
			: typeof value.amount === "number"
				? value.amount
				: null;
		const liquidValue = liquidAmount !== null ? currencyFormatter.format(liquidAmount) : noValueMarkup;
		const totalValue = totalAmount !== null ? currencyFormatter.format(totalAmount) : noValueMarkup;

		tr.appendChild(createCell(liquidValue, "amount-cell"));
		tr.appendChild(createCell(totalValue, "amount-cell"));
		tr.appendChild(createCell(formatNotes(value.notes)));

		tableBody.appendChild(tr);
	});
}

function prepareDownload(csvContent) {
	if (!downloadLink) {
		return;
	}
	if (currentObjectUrl) {
		URL.revokeObjectURL(currentObjectUrl);
		currentObjectUrl = null;
	}

	if (!csvContent) {
		latestCsvContent = "";
		latestCsvFilename = "";
		downloadLink.classList.add("hidden");
		return;
	}

	latestCsvContent = csvContent;
	latestCsvFilename = `forecast-${new Date().toISOString().split("T")[0]}.csv`;
	downloadLink.classList.remove("hidden");
}

async function handleCsvDownloadClick() {
	if (!downloadLink) {
		return;
	}
	if (!latestCsvContent) {
		showMessage("Run a forecast to generate results before downloading.", "error");
		return;
	}

	const filename = latestCsvFilename || `forecast-${new Date().toISOString().split("T")[0]}.csv`;
	const blob = new Blob([latestCsvContent], { type: "text/csv" });

	const result = await saveBlobWithPickerOrFallback(blob, {
		suggestedName: filename,
		mimeType: "text/csv",
		extensions: [".csv"],
		description: "Forecast results (CSV)",
		fallbackDownload: () => {
			if (currentObjectUrl) {
				URL.revokeObjectURL(currentObjectUrl);
			}
			currentObjectUrl = URL.createObjectURL(blob);
			triggerAnchorDownload(currentObjectUrl, filename);
		},
	});

	if (result === "saved") {
		showMessage("Forecast CSV saved to your chosen location.", "success");
	} else if (result === "fallback") {
		showMessage("Forecast CSV downloaded to your device.", "success");
	} else if (result === "cancelled") {
		showMessage("CSV download canceled.", null);
	} else if (result === "unavailable") {
		showMessage("Downloading is not supported in this browser.", "error");
	} else if (result === "error") {
		showMessage("Unable to download the CSV file. Please try again.", "error");
	}
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
	common.investments = Array.isArray(common.investments)
		? common.investments.map(normalizeInvestment)
		: [];

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

function normalizeInvestment(investment) {
	const normalized = cloneDeep(investment) || {};
	delete normalized.dateList;
	delete normalized.DateList;
	normalized.contributions = Array.isArray(normalized.contributions)
		? normalized.contributions.map(normalizeEvent)
		: [];
	normalized.withdrawals = Array.isArray(normalized.withdrawals)
		? normalized.withdrawals.map(normalizeEvent)
		: [];
	if (typeof normalized.contributionsFromCash !== "boolean") {
		normalized.contributionsFromCash = Boolean(normalized.contributionsFromCash);
	}
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
	normalized.investments = Array.isArray(normalized.investments)
		? normalized.investments.map(normalizeInvestment)
		: [];
	if (typeof normalized.active !== "boolean") {
		normalized.active = Boolean(normalized.active);
	}
	return normalized;
}

function renderConfigEditor() {
	closeActiveHelpTooltip();
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
		enableNowShortcut: true,
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
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Balance at the end of the start month. Calculate this as your liquid net worth: cash and cash-equivalents minus short-term debts (e.g., credit card balances).",
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
	commonSection.body.appendChild(createInvestmentCollection(currentConfig.common.investments, "common.investments", {
		heading: "Common investments",
		addLabel: "Add common investment",
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

	card.appendChild(createInvestmentCollection(scenario.investments || [], `scenarios[${index}].investments`, {
		heading: "Scenario investments",
		addLabel: "Add investment",
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
			const card = createEventCard(event, `${basePath}[${index}]`, index, options, () => {
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
		const factory = typeof options.createEmptyEvent === "function" ? options.createEmptyEvent : createEmptyEvent;
		events.push(factory());
		renderConfigEditor();
		switchTab("config");
	});
	actions.appendChild(addButton);
	container.appendChild(actions);

	return container;
}

function createEventCard(event, basePath, index, options = {}, onRemove) {
	const { titlePrefix = "Event", enableWithdrawalPercentage = false } = options || {};

	const card = document.createElement("div");
	card.className = "editor-card";

	const { header, title } = createCardHeader(
		event.name || `${titlePrefix} ${index + 1}`,
		onRemove,
		options.removeLabel || "Remove event",
	);
	card.appendChild(header);

	const grid = document.createElement("div");
	grid.className = "editor-grid";

	const amountPath = `${basePath}.amount`;
	const percentagePath = `${basePath}.percentage`;
	let modeSelect;
	let applyMode = () => {};

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

	if (enableWithdrawalPercentage) {
		const modeField = document.createElement("label");
		modeField.className = "editor-field select-field";
		const modeLabel = document.createElement("span");
		modeLabel.className = "editor-label";
		modeLabel.textContent = "Withdrawal type";
		modeField.appendChild(modeLabel);
		attachFieldHelp({
			wrapper: modeField,
			labelEl: modeLabel,
			tooltipText: "Choose whether this withdrawal uses a fixed dollar amount or a percentage of the current investment balance.",
			label: "Withdrawal type",
		});

		modeSelect = document.createElement("select");
		modeSelect.innerHTML = `
			<option value="amount">Fixed amount</option>
			<option value="percentage">Percentage of balance</option>
		`;
		modeSelect.addEventListener("change", () => {
			if (modeSelect && typeof modeSelect.value === "string") {
				applyMode(modeSelect.value);
			}
		});
		modeField.appendChild(modeSelect);
		grid.appendChild(modeField);
	}

	const defaultAmountTooltip = enableWithdrawalPercentage
		? "Fixed dollar amount withdrawn when this event occurs."
		: "Positive amounts represent income; negative amounts represent expenses.";
	const amountTooltip = options.amountTooltip || defaultAmountTooltip;

	const amountField = createInputField({
		label: enableWithdrawalPercentage ? "Amount (USD)" : "Amount",
		path: amountPath,
		value: event.amount ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_LARGE,
		tooltip: amountTooltip,
		validation: { type: "number" },
	});
	grid.appendChild(amountField);

	let percentageField = null;
	if (enableWithdrawalPercentage) {
		percentageField = createInputField({
			label: "Percentage (%)",
			path: percentagePath,
			value: event.percentage ?? "",
			inputType: "number",
			step: "0.01",
			arrowStep: ARROW_STEP_SMALL,
			min: 0,
			tooltip: "Percentage of the investment balance withdrawn when this event occurs.",
			validation: { type: "number" },
		});
		grid.appendChild(percentageField);
	}
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
		tooltip: "Optional month when this event begins (YYYY-MM). Defaults to the simulation start month when left blank.",
		validation: { type: "month" },
		maxLength: 7,
		enableNowShortcut: true,
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

	if (enableWithdrawalPercentage) {
		const hasPercentage = Object.prototype.hasOwnProperty.call(event, "percentage") && event.percentage !== undefined;
		const hasAmount = Object.prototype.hasOwnProperty.call(event, "amount") && event.amount !== undefined;
		let currentMode = hasPercentage && !hasAmount ? "percentage" : "amount";
		if (hasPercentage && event.percentage !== 0) {
			currentMode = "percentage";
		}

		applyMode = (mode) => {
			const normalizedMode = mode === "percentage" ? "percentage" : "amount";
			if (modeSelect) {
				modeSelect.value = normalizedMode;
			}

			setFieldDisabled(amountPath, normalizedMode !== "amount");
			setFieldDisabled(percentagePath, normalizedMode !== "percentage");

			if (normalizedMode === "amount") {
				updateConfigAtPath(percentagePath, null, "number");
				setFieldValue(percentagePath, "");
				delete event.percentage;
				if (!Object.prototype.hasOwnProperty.call(event, "amount")) {
					updateConfigAtPath(amountPath, 0, "number");
					setFieldValue(amountPath, "0");
					event.amount = 0;
				}
			} else {
				updateConfigAtPath(amountPath, null, "number");
				setFieldValue(amountPath, "");
				delete event.amount;
				if (!Object.prototype.hasOwnProperty.call(event, "percentage")) {
					updateConfigAtPath(percentagePath, 0, "number");
					setFieldValue(percentagePath, "0");
					event.percentage = 0;
				}
			}

			updateEditorActionsState();
		};

		applyMode(currentMode);
	}
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

function createInvestmentCollection(investments, basePath, options = {}) {
	const container = document.createElement("div");
	container.className = "editor-subsection";

	if (options.heading) {
		const heading = document.createElement("h4");
		heading.textContent = options.heading;
		container.appendChild(heading);
	}

	const collection = document.createElement("div");
	collection.className = "editor-collection";

	if (!Array.isArray(investments)) {
		investments = [];
	}

	if (investments.length === 0) {
		const emptyState = document.createElement("p");
		emptyState.className = "muted-text";
		emptyState.textContent = options.emptyMessage || "No investments configured.";
		collection.appendChild(emptyState);
	} else {
		investments.forEach((investment, index) => {
			const card = createInvestmentCard(investment, `${basePath}[${index}]`, index, options.titlePrefix || "Investment", () => {
				investments.splice(index, 1);
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
	addButton.textContent = options.addLabel || "Add investment";
	addButton.addEventListener("click", () => {
		investments.push(createEmptyInvestment());
		renderConfigEditor();
		switchTab("config");
	});
	actions.appendChild(addButton);
	container.appendChild(actions);

	return container;
}

function createInvestmentCard(investment, basePath, index, titlePrefix, onRemove) {
	const card = document.createElement("div");
	card.className = "editor-card";

	if (!Array.isArray(investment.contributions)) {
		investment.contributions = [];
	}
	if (!Array.isArray(investment.withdrawals)) {
		investment.withdrawals = [];
	}

	const { header, title } = createCardHeader(
		investment.name || `${titlePrefix} ${index + 1}`,
		onRemove,
		"Remove investment",
	);
	card.appendChild(header);

	const grid = document.createElement("div");
	grid.className = "editor-grid";
	grid.appendChild(createInputField({
		label: "Name",
		path: `${basePath}.name`,
		value: investment.name ?? "",
		inputType: "text",
		tooltip: "Optional label displayed in reports for this investment.",
		validation: { type: "text", maxLength: 120 },
		maxLength: 120,
		onChange: (value) => {
			title.textContent = value || `${titlePrefix} ${index + 1}`;
		},
	}));
	grid.appendChild(createInputField({
		label: "Starting value",
		path: `${basePath}.startingValue`,
		value: investment.startingValue ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Current balance of the investment at the start date.",
		validation: { type: "number" },
	}));
	grid.appendChild(createInputField({
		label: "Annual return rate (%)",
		path: `${basePath}.annualReturnRate`,
		value: investment.annualReturnRate ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_SMALL,
		tooltip: "Expected average annual rate of return expressed as a percentage.",
		validation: { type: "number" },
	}));
	grid.appendChild(createInputField({
		label: "Tax rate on gains (%)",
		path: `${basePath}.taxRate`,
		value: investment.taxRate ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_SMALL,
		tooltip: "Optional tax rate applied to positive monthly gains.",
		validation: { type: "number", min: 0, max: 100 },
	}));
	grid.appendChild(createCheckboxField({
		label: "Contributions reduce cash balance",
		path: `${basePath}.contributionsFromCash`,
		value: investment.contributionsFromCash,
		tooltip: "Enable when contribution amounts should be deducted from your simulated cash balance (e.g. Roth IRA or taxable brokerage). Disable for pre-tax payroll deductions such as traditional 401(k).",
	}));
	card.appendChild(grid);

	card.appendChild(createEventCollection(investment.contributions, `${basePath}.contributions`, {
		heading: "Contributions",
		titlePrefix: "Contribution",
		addLabel: "Add contribution",
		emptyMessage: "No contributions scheduled.",
		amountTooltip: "Amount contributed each time this event occurs. Enter a positive value; contributions increase this investment's balance.",
	}));

	card.appendChild(createEventCollection(investment.withdrawals, `${basePath}.withdrawals`, {
		heading: "Withdrawals",
		titlePrefix: "Withdrawal",
		addLabel: "Add withdrawal",
		emptyMessage: "No withdrawals scheduled.",
		enableWithdrawalPercentage: true,
		createEmptyEvent: createEmptyWithdrawalEvent,
	}));

	return card;
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
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Original loan principal before any down payment is applied.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Down payment",
		path: `${basePath}.downPayment`,
		value: loan.downPayment ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Amount paid up front to reduce the principal.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Interest rate (%)",
		path: `${basePath}.interestRate`,
		value: loan.interestRate ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_SMALL,
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
		tooltip: "Required month when this loan begins (YYYY-MM). Loans do not assume a default start month.",
		validation: { type: "month" },
		maxLength: 7,
		enableNowShortcut: true,
	}));
	grid.appendChild(createInputField({
		label: "Escrow",
		path: `${basePath}.escrow`,
		value: loan.escrow ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Optional monthly escrow payment associated with the loan.",
		validation: { type: "number" },
	}));
	grid.appendChild(createInputField({
		label: "Mortgage insurance",
		path: `${basePath}.mortgageInsurance`,
		value: loan.mortgageInsurance ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Monthly mortgage insurance premium, if applicable.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Mortgage insurance cutoff (%)",
		path: `${basePath}.mortgageInsuranceCutoff`,
		value: loan.mortgageInsuranceCutoff ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_SMALL,
		tooltip: "Loan-to-value percentage at which mortgage insurance ends.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Early payoff threshold",
		path: `${basePath}.earlyPayoffThreshold`,
		value: loan.earlyPayoffThreshold ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Amount of cash you want to have remaining if you choose to pay off the loan early.",
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
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Expected sale price when the property is sold.",
		validation: { type: "number", min: 0 },
	}));
	grid.appendChild(createInputField({
		label: "Sell costs (net)",
		path: `${basePath}.sellCostsNet`,
		value: loan.sellCostsNet ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_LARGE,
		tooltip: "Net costs (positive) or proceeds (negative) incurred when selling.",
		validation: { type: "number" },
	}));
	card.appendChild(grid);

	const extraPayments = createEventCollection(loan.extraPrincipalPayments, `${basePath}.extraPrincipalPayments`, {
		heading: "Extra principal payments",
		titlePrefix: "Payment",
		addLabel: "Add extra payment",
		emptyMessage: "No extra principal payments configured.",
		amountTooltip: "Extra payment applied directly to the loan principal each time this event occurs. Enter a positive value to reduce the balance faster.",
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

function initHelpTooltipSystem() {
	if (helpTooltipInitialized) {
		return;
	}
	helpTooltipInitialized = true;

	document.addEventListener("click", (event) => {
		if (!activeHelpTooltip) {
			return;
		}
		const { trigger, tooltip } = activeHelpTooltip;
		if (trigger && trigger.contains(event.target)) {
			return;
		}
		if (tooltip && tooltip.contains(event.target)) {
			return;
		}
		closeActiveHelpTooltip();
	});

	document.addEventListener("keydown", (event) => {
		if (event.key !== "Escape" || !activeHelpTooltip) {
			return;
		}
		const { trigger } = activeHelpTooltip;
		closeActiveHelpTooltip();
		if (trigger && typeof trigger.focus === "function") {
			trigger.focus();
		}
	});
}

function openHelpTooltip(trigger, tooltip) {
	if (!trigger || !tooltip) {
		return;
	}
	if (activeHelpTooltip && activeHelpTooltip.trigger === trigger) {
		return;
	}
	closeActiveHelpTooltip();
	tooltip.classList.remove("hidden");
	trigger.setAttribute("aria-expanded", "true");
	trigger.setAttribute("aria-describedby", tooltip.id);
	trigger.classList.add("active");
	activeHelpTooltip = { trigger, tooltip };
}

function closeActiveHelpTooltip() {
	if (!activeHelpTooltip) {
		return;
	}
	const { trigger, tooltip } = activeHelpTooltip;
	if (trigger) {
		trigger.setAttribute("aria-expanded", "false");
		trigger.removeAttribute("aria-describedby");
		trigger.classList.remove("active");
	}
	if (tooltip) {
		tooltip.classList.add("hidden");
	}
	activeHelpTooltip = null;
}

function attachFieldHelp({ wrapper, labelEl, tooltipText, label }) {
	if (!tooltipText) {
		return null;
	}

	initHelpTooltipSystem();

	tooltipCounter += 1;
	const tooltipId = `field-tooltip-${tooltipCounter}`;
	const accessibleLabel = label ? `${label} help` : "Field information";

	const trigger = document.createElement("button");
	trigger.type = "button";
	trigger.className = "field-help-button";
	trigger.setAttribute("aria-expanded", "false");
	trigger.setAttribute("aria-controls", tooltipId);
	trigger.setAttribute("aria-label", accessibleLabel);
	trigger.innerHTML = "<span aria-hidden=\"true\">i</span>";

	const tooltip = document.createElement("span");
	tooltip.className = "field-tooltip hidden";
	tooltip.id = tooltipId;
	tooltip.setAttribute("role", "tooltip");
	tooltip.textContent = tooltipText;

	trigger.addEventListener("click", (event) => {
		event.preventDefault();
		event.stopPropagation();
		if (activeHelpTooltip && activeHelpTooltip.trigger === trigger) {
			closeActiveHelpTooltip();
		} else {
			openHelpTooltip(trigger, tooltip);
		}
	});

	trigger.addEventListener("pointerdown", (event) => {
		event.stopPropagation();
	});

	trigger.addEventListener("keydown", (event) => {
		if (event.key === "Escape" && activeHelpTooltip && activeHelpTooltip.trigger === trigger) {
			closeActiveHelpTooltip();
		}
	});

	tooltip.addEventListener("click", (event) => {
		event.stopPropagation();
	});

	labelEl.appendChild(trigger);
	wrapper.appendChild(tooltip);

	return { trigger, tooltip };
}

function deriveArrowPrecision(rawStep) {
	if (!Number.isFinite(rawStep) || rawStep <= 0) {
		return null;
	}
	const stepString = rawStep.toString().toLowerCase();
	const scientificMatch = stepString.match(/e-(\d+)$/);
	if (scientificMatch) {
		return Number.parseInt(scientificMatch[1], 10) || 0;
	}
	const decimalIndex = stepString.indexOf(".");
	return decimalIndex >= 0 ? stepString.length - decimalIndex - 1 : 0;
}

function computeArrowPrecision(step, numberKind) {
	let precision = null;
	if (typeof step === "number") {
		precision = deriveArrowPrecision(step);
	} else if (typeof step === "string") {
		const parsedStep = Number(step);
		if (Number.isFinite(parsedStep)) {
			precision = deriveArrowPrecision(parsedStep);
		}
	}
	if (precision === null && numberKind === "int") {
		return 0;
	}
	return precision;
}

function resolveNumericBounds(min, validation) {
	const minValue = typeof min === "number"
		? min
		: validation && typeof validation.min === "number"
			? validation.min
			: undefined;
	const maxValue = validation && typeof validation.max === "number"
		? validation.max
		: undefined;

	return { minValue, maxValue };
}

function clampValue(value, minValue, maxValue) {
	let result = value;
	if (minValue !== undefined && result < minValue) {
		result = minValue;
	}
	if (maxValue !== undefined && result > maxValue) {
		result = maxValue;
	}
	return result;
}

function setupArrowKeyStep(control, options) {
	const { arrowStep, step, numberKind, min, validation } = options;
	if (typeof arrowStep !== "number" || !Number.isFinite(arrowStep) || arrowStep === 0) {
		return;
	}

	const precision = computeArrowPrecision(step, numberKind);
	const { minValue, maxValue } = resolveNumericBounds(min, validation);

	control.addEventListener("keydown", (event) => {
		if (event.key !== "ArrowUp" && event.key !== "ArrowDown") {
			return;
		}
		if (event.shiftKey || event.altKey || event.ctrlKey || event.metaKey) {
			return;
		}
		if (control.disabled || control.readOnly) {
			return;
		}
		event.preventDefault();

		const rawValue = control.value === "" ? null : Number(control.value);
		const currentValue = Number.isFinite(rawValue) ? rawValue : 0;
		const direction = event.key === "ArrowUp" ? 1 : -1;
		let nextValue = currentValue + direction * arrowStep;
		nextValue = clampValue(nextValue, minValue, maxValue);

		let formattedValue;
		if (numberKind === "int") {
			nextValue = Math.trunc(nextValue);
			formattedValue = String(nextValue);
		} else if (precision !== null && precision >= 0) {
			formattedValue = nextValue.toFixed(precision);
		} else {
			formattedValue = String(nextValue);
		}

		control.value = formattedValue;
		control.dispatchEvent(new Event("input", { bubbles: true }));
	});
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
	disabled = false,
	arrowStep,
	enableNowShortcut = false,
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

	const addonButtons = [];
	let controlMount = control;
	const shouldAttachNowShortcut = enableNowShortcut && inputType === "month";

	if (shouldAttachNowShortcut) {
		wrapper.classList.add("field-with-addon");
		const group = document.createElement("div");
		group.className = "editor-input-with-addon";
		group.appendChild(control);

		const nowButton = document.createElement("button");
		nowButton.type = "button";
		nowButton.className = "field-inline-button";
		nowButton.textContent = "Now";
		nowButton.title = "Set to the current month";
		nowButton.setAttribute("aria-label", "Set to the current month");
		nowButton.addEventListener("click", () => {
			const currentMonth = getCurrentMonthValue();
			if (control.value !== currentMonth) {
				control.value = currentMonth;
				control.dispatchEvent(new Event("input", { bubbles: true }));
			}
			try {
				control.focus({ preventScroll: true });
			} catch (error) {
				control.focus();
			}
		});
		addonButtons.push(nowButton);
		group.appendChild(nowButton);
		controlMount = group;
	}

	if (typeof maxLength === "number" && control.tagName === "INPUT") {
		control.maxLength = maxLength;
	}

	if (tooltip) {
		control.title = tooltip;
	}

	if (validation && validation.type === "month") {
		control.setAttribute("pattern", "\\d{4}-(0[1-9]|1[0-2])");
	}

	control.dataset.path = path;
	attachFieldHelp({ wrapper, labelEl, tooltipText: tooltip, label });

	if (disabled) {
		control.disabled = true;
		wrapper.classList.add("is-disabled");
		addonButtons.forEach((button) => {
			button.disabled = true;
		});
	}

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
		disabled,
		addonButtons,
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

	if (inputType === "number") {
		setupArrowKeyStep(control, {
			arrowStep,
			step,
			numberKind,
			min,
			validation,
		});
	}

	registeredInputs.push(entry);
	if (entry.validation && !disabled) {
		runFieldValidation(entry, { report: false });
	} else {
		entry.isValid = true;
	}

	wrapper.appendChild(controlMount);
	wrapper.appendChild(errorEl);
	return wrapper;
}

function findRegisteredInput(path) {
	if (!path) {
		return null;
	}
	return registeredInputs.find((entry) => entry.path === path) || null;
}

function setFieldDisabled(path, disabled) {
	const entry = findRegisteredInput(path);
	if (!entry) {
		return;
	}

	entry.disabled = disabled;
	entry.control.disabled = disabled;

	const wrapper = entry.control.closest(".editor-field");
	if (wrapper) {
		wrapper.classList.toggle("is-disabled", disabled);
	}

	if (Array.isArray(entry.addonButtons)) {
		entry.addonButtons.forEach((button) => {
			button.disabled = disabled;
		});
	}

	if (disabled) {
		entry.isValid = true;
		entry.errorEl.classList.add("hidden");
		entry.control.classList.remove("invalid");
		entry.control.removeAttribute("aria-invalid");
	} else if (entry.validation) {
		runFieldValidation(entry, { report: false });
	}
}

function setFieldValue(path, value) {
	const entry = findRegisteredInput(path);
	if (!entry) {
		return;
	}

	const control = entry.control;
	if (value === null || value === undefined) {
		control.value = "";
	} else if (typeof value === "number") {
		control.value = Number.isFinite(value) ? String(value) : "";
	} else {
		control.value = String(value);
	}
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
	}

	const labelEl = document.createElement("span");
	labelEl.className = "editor-label";
	labelEl.textContent = label;
	attachFieldHelp({ wrapper, labelEl, tooltipText: tooltip, label });

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
	if (entry.disabled) {
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

function createEmptyWithdrawalEvent() {
	return {
		name: "",
		percentage: 0,
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

function createEmptyInvestment() {
	return {
		name: "",
		startingValue: 0,
		annualReturnRate: 0,
		taxRate: 0,
		contributions: [],
		withdrawals: [],
		contributionsFromCash: false,
	};
}

function createEmptyScenario() {
	return {
		name: "",
		active: true,
		events: [],
		loans: [],
		investments: [],
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
			investments: [],
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

	closeActiveHelpTooltip();

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
		const result = await triggerConfigDownload(latestConfigYaml);
		if (result === "saved") {
			showMessage("Configuration saved to your chosen location.", "success");
		} else if (result === "fallback") {
			showMessage("Configuration downloaded to your device.", "success");
		} else if (result === "cancelled") {
			showMessage("Configuration download canceled.", null);
		} else {
			showMessage("Unable to download the configuration. Please try again.", "error");
		}
	} catch (error) {
		console.error("Download config failed", error);
		showMessage(error.message, "error");
	} finally {
		const hasConfig = !!currentConfig;
		downloadConfigButton.disabled = !hasConfig;
		runForecastButton.disabled = !hasConfig;
	}
}

async function triggerConfigDownload(yamlContent) {
	if (!yamlContent) {
		return "unavailable";
	}

	const filename = `config-${new Date().toISOString().split("T")[0]}.yaml`;
	const blob = new Blob([yamlContent], { type: "text/yaml" });

	const result = await saveBlobWithPickerOrFallback(blob, {
		suggestedName: filename,
		mimeType: "text/yaml",
		extensions: [".yaml", ".yml"],
		description: "Finance Forecast configuration",
		fallbackDownload: () => {
			if (configDownloadUrl) {
				URL.revokeObjectURL(configDownloadUrl);
			}
			configDownloadUrl = URL.createObjectURL(blob);
			triggerAnchorDownload(configDownloadUrl, filename);
		},
	});

	return result;
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
