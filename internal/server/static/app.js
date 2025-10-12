const messageEl = document.getElementById("message");
const warningsEl = document.getElementById("warnings");
const workspacePanel = document.getElementById("workspace-panel");
const resultsPanel = document.getElementById("results-section");
const configPanel = document.getElementById("config-panel");
const tableHead = document.querySelector("#results-table thead");
const tableBody = document.querySelector("#results-table tbody");
const downloadLink = document.getElementById("download-link");
const durationEl = document.getElementById("duration");
const scenarioTabsEl = document.getElementById("scenario-tabs");
const chartWrapper = document.getElementById("results-chart-wrapper");
const chartSvg = document.getElementById("results-chart");
const chartLegendEl = document.getElementById("results-chart-legend");
const chartEmptyEl = document.getElementById("results-chart-empty");
const chartTitleEl = document.getElementById("results-chart-title");
const chartCaptionEl = document.getElementById("results-chart-caption");
const resultsSummaryEl = document.getElementById("results-summary");
const chartTooltipEl = document.getElementById("results-chart-tooltip");
const chartTooltipDateEl = chartTooltipEl ? chartTooltipEl.querySelector('[data-role="tooltip-date"]') : null;
const chartTooltipLiquidEl = chartTooltipEl ? chartTooltipEl.querySelector('[data-role="tooltip-liquid"]') : null;
const chartTooltipTotalEl = chartTooltipEl ? chartTooltipEl.querySelector('[data-role="tooltip-total"]') : null;
const configEditorRoot = document.getElementById("config-editor");
const uploadConfigInput = document.getElementById("upload-config-input");
const uploadConfigButton = document.getElementById("upload-config-button");
const runForecastButton = document.getElementById("run-forecast-button");
const downloadConfigButton = document.getElementById("download-config-button");
const resetConfigButton = document.getElementById("reset-config-button");
const editorLoading = document.getElementById("editor-loading");
const tablistContainer = document.querySelector(".tablist-container");
const versionFooter = document.getElementById("workspace-footer");
const versionLabel = document.getElementById("app-version-label");
const optimizerToggleInput = document.getElementById("optimizer-toggle-input");
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
let editorPersistTimer = null;
let editorStorageAvailable = null;
let editorPersistenceHandlersRegistered = false;
let chartResizeFrame = null;
let stickyInlineErrorEl = null;
let stickyInlineErrorAnchor = null;
const THEME_STORAGE_KEY = "financeForecast.theme";
const EDITOR_STORAGE_KEY = "financeForecast.editorState.v1";
const EDITOR_STORAGE_VERSION = 1;
const EDITOR_PERSIST_DEBOUNCE_MS = 600;
const OPTIMIZER_STORAGE_KEY = "financeForecast.optimizerEnabled.v1";

let optimizerEnabled = false;

function loadOptimizerPreference() {
	if (typeof window === "undefined" || !window.localStorage) {
		return false;
	}

	try {
		const rawValue = window.localStorage.getItem(OPTIMIZER_STORAGE_KEY);
		if (rawValue === null) {
			return false;
		}
		return rawValue === "1" || rawValue === "true";
	} catch (error) {
		console.warn("Unable to read optimizer preference from storage.", error);
		return false;
	}
}

function persistOptimizerPreference(enabled) {
	if (typeof window === "undefined" || !window.localStorage) {
		return;
	}

	try {
		window.localStorage.setItem(OPTIMIZER_STORAGE_KEY, enabled ? "1" : "0");
	} catch (error) {
		console.warn("Unable to persist optimizer preference.", error);
	}
}

function updateOptimizerToggleUI() {
	if (!optimizerToggleInput) {
		return;
	}

	optimizerToggleInput.checked = Boolean(optimizerEnabled);
	optimizerToggleInput.setAttribute("aria-pressed", optimizerEnabled ? "true" : "false");
	optimizerToggleInput.setAttribute("aria-label", optimizerEnabled ? "Disable optimizer" : "Enable optimizer");
	rootElement.classList.toggle("optimizer-enabled", optimizerEnabled);
}

function setOptimizerEnabledState(enabled, options = {}) {
	const normalized = Boolean(enabled);
	const { skipRender = false } = options || {};
	if (optimizerEnabled === normalized) {
		updateOptimizerToggleUI();
		return;
	}

	optimizerEnabled = normalized;
	persistOptimizerPreference(optimizerEnabled);
	updateOptimizerToggleUI();

	if (!skipRender) {
		renderConfigEditor();
	}
}

function initializeOptimizerControls() {
	setOptimizerEnabledState(false, { skipRender: true });

	if (!optimizerToggleInput) {
		return;
	}

	optimizerToggleInput.addEventListener("change", () => {
		setOptimizerEnabledState(optimizerToggleInput.checked);
	});
}

const MONTH_PATTERN = /^\d{4}-(0[1-9]|1[0-2])$/;
const SVG_NS = "http://www.w3.org/2000/svg";
const CHART_MARGIN = {
	top: 44,
	right: 36,
	bottom: 90,
	left: 96,
};
const CHART_MIN_HEIGHT = 220;
const CHART_MAX_HEIGHT = 360;
const CHART_ASPECT_RATIO = 0.55;
const CHART_SERIES = [
	{ key: "liquid", label: "Liquid Net Worth", lineClass: "chart-line--liquid", pointClass: "chart-point--liquid", swatchClass: "chart-legend-swatch--liquid" },
	{ key: "total", label: "Total Net Worth", lineClass: "chart-line--total", pointClass: "chart-point--total", swatchClass: "chart-legend-swatch--total" },
];
const SUMMARY_CURRENCY_FORMATTER = new Intl.NumberFormat(undefined, {
	style: "currency",
	currency: "USD",
	minimumFractionDigits: 2,
	maximumFractionDigits: 2,
});
const CHART_TOOLTIP_CURRENCY_FORMATTER = new Intl.NumberFormat(undefined, {
	style: "currency",
	currency: "USD",
	minimumFractionDigits: 2,
	maximumFractionDigits: 2,
});
const CHART_TOOLTIP_DATE_FORMATTER = new Intl.DateTimeFormat(undefined, {
	month: "long",
	year: "numeric",
});

const OPTIMIZER_FIELD_OPTIONS = [
	{ label: "Amount", value: "amount" },
	{ label: "Frequency", value: "frequency" },
	{ label: "Start date", value: "startDate" },
	{ label: "End date", value: "endDate" },
];

const OPTIMIZER_FIELD_DESCRIPTIONS = {
	amount: "Adjust this event's amount to keep cash at or above the emergency-fund floor once achieved.",
	frequency: "Adjust how often this event recurs to help maintain the emergency-fund floor.",
	startDate: "Adjust when this event begins to align cash flow with the emergency-fund floor.",
	endDate: "Adjust when this event ends to maintain the emergency-fund floor.",
};

const NET_WORTH_METRIC_LABELS = {
	total: "Total net worth",
	liquid: "Liquid net worth",
	amount: "Net worth",
};

const OPTIMIZER_FIELD_CANONICAL = {
	amount: "amount",
	frequency: "frequency",
	startdate: "startDate",
	"start-date": "startDate",
	start_date: "startDate",
	enddate: "endDate",
	"end-date": "endDate",
	end_date: "endDate",
};

function formatSummaryCurrency(value) {
	return typeof value === "number" && Number.isFinite(value) ? SUMMARY_CURRENCY_FORMATTER.format(value) : null;
}

function getNetWorthMetricLabel(metric) {
	if (typeof metric !== "string" || metric === "") {
		return NET_WORTH_METRIC_LABELS.amount;
	}
	return NET_WORTH_METRIC_LABELS[metric] || NET_WORTH_METRIC_LABELS.amount;
}

function getNegativeNetWorthValue(value) {
	if (!value || typeof value !== "object") {
		return null;
	}
	const total = typeof value.total === "number" && Number.isFinite(value.total) ? value.total : null;
	if (total !== null && total < 0) {
		return { value: total, metric: "total" };
	}
	const liquid = typeof value.liquid === "number" && Number.isFinite(value.liquid) ? value.liquid : null;
	if (liquid !== null && liquid < 0) {
		return { value: liquid, metric: "liquid" };
	}
	const amount = typeof value.amount === "number" && Number.isFinite(value.amount) ? value.amount : null;
	if (amount !== null && amount < 0) {
		return { value: amount, metric: "amount" };
	}
	return null;
}

function findNegativeNetWorthSpans(rows, scenarioIndex) {
	const spans = [];
	if (!Array.isArray(rows) || rows.length === 0 || typeof scenarioIndex !== "number" || scenarioIndex < 0) {
		return spans;
	}

	let activeSpan = null;
	rows.forEach((row, index) => {
		const value = Array.isArray(row?.values) ? row.values[scenarioIndex] || {} : {};
		const candidate = getNegativeNetWorthValue(value);
		if (candidate) {
			const { value: negativeValue, metric } = candidate;
			const dateLabel = typeof row?.date === "string" && row.date.trim() !== "" ? row.date.trim() : `Month ${index + 1}`;
			if (!activeSpan) {
				activeSpan = {
					startIndex: index,
					endIndex: index,
					startDate: dateLabel,
					endDate: dateLabel,
					length: 1,
					minValue: negativeValue,
					metric,
				};
			} else {
				activeSpan.endIndex = index;
				activeSpan.endDate = dateLabel;
				activeSpan.length += 1;
				if (negativeValue < activeSpan.minValue) {
					activeSpan.minValue = negativeValue;
				}
				if (metric === "total" && activeSpan.metric !== "total") {
					activeSpan.metric = "total";
				}
			}
		} else if (activeSpan) {
			spans.push(activeSpan);
			activeSpan = null;
		}
	});

	if (activeSpan) {
		spans.push(activeSpan);
	}

	return spans;
}

function calculateNegativeNetWorthSegments(points) {
	const segments = [];
	if (!Array.isArray(points) || points.length === 0) {
		return segments;
	}

	let start = -1;
	let segmentMetric = null;
	points.forEach((point, index) => {
		const candidate = getNegativeChartValue(point);
		if (candidate) {
			if (start === -1) {
				start = index;
				segmentMetric = candidate.metric;
			} else if (candidate.metric === "total" && segmentMetric !== "total") {
				segmentMetric = "total";
			}
		} else if (start !== -1) {
			segments.push({ startIndex: start, endIndex: index - 1, metric: segmentMetric });
			start = -1;
			segmentMetric = null;
		}
	});

	if (start !== -1) {
		segments.push({ startIndex: start, endIndex: points.length - 1, metric: segmentMetric });
	}

	return segments;
}

function getNegativeChartValue(point) {
	if (!point || typeof point !== "object") {
		return null;
	}
	const total = typeof point.total === "number" && Number.isFinite(point.total) ? point.total : null;
	if (total !== null && total < 0) {
		return { value: total, metric: "total" };
	}
	const liquid = typeof point.liquid === "number" && Number.isFinite(point.liquid) ? point.liquid : null;
	if (liquid !== null && liquid < 0) {
		return { value: liquid, metric: "liquid" };
	}
	return null;
}

function calculateSegmentDomainBounds(points, startIndex, endIndex) {
	if (!Array.isArray(points) || points.length === 0) {
		return null;
	}
	if (typeof startIndex !== "number" || typeof endIndex !== "number" || startIndex < 0 || endIndex < startIndex || endIndex >= points.length) {
		return null;
	}

	const startPoint = points[startIndex];
	const endPoint = points[endIndex];
	if (!startPoint || !endPoint) {
		return null;
	}

	let startTime = startPoint.time;
	let endTime = endPoint.time;

	if (startIndex > 0 && Number.isFinite(points[startIndex - 1]?.time) && Number.isFinite(startTime)) {
		startTime = (points[startIndex - 1].time + startTime) / 2;
	}
	if (endIndex < points.length - 1 && Number.isFinite(points[endIndex + 1]?.time) && Number.isFinite(endTime)) {
		endTime = (endPoint.time + points[endIndex + 1].time) / 2;
	}

	return {
		startTime,
		endTime,
	};
}

function formatOptimizerDisplay(summary, kind) {
	if (!summary || (kind !== "original" && kind !== "value")) {
		return null;
	}

	if (kind === "original" && typeof summary.originalDisplay === "string" && summary.originalDisplay.trim() !== "") {
		return summary.originalDisplay.trim();
	}
	if (kind === "value" && typeof summary.valueDisplay === "string" && summary.valueDisplay.trim() !== "") {
		return summary.valueDisplay.trim();
	}

	const field = typeof summary.field === "string" ? summary.field.toLowerCase() : "amount";
	const numeric = kind === "original" ? summary.original : summary.value;
	const numericValue = typeof numeric === "number" && Number.isFinite(numeric) ? numeric : Number(numeric);

	if (!Number.isFinite(numericValue)) {
		return null;
	}

	if (field === "" || field === "amount") {
		return formatSummaryCurrency(numericValue);
	}
	if (field === "frequency") {
		const rounded = Math.round(numericValue);
		return Number.isFinite(rounded) ? String(rounded) : null;
	}
	if (field === "startdate" || field === "enddate") {
		return formatMonthIndexValue(numericValue);
	}

	return String(numericValue);
}

function formatMonthIndexValue(value) {
	const index = Math.round(value);
	if (!Number.isFinite(index)) {
		return null;
	}
	const year = Math.trunc(index / 12);
	const month = (index % 12) + 1;
	if (!Number.isFinite(year) || !Number.isFinite(month)) {
		return null;
	}
	return `${String(year).padStart(4, "0")}-${String(month).padStart(2, "0")}`;
}

function normalizeOptimizerField(field) {
	const key = typeof field === "string" ? field.trim().toLowerCase() : "amount";
	return OPTIMIZER_FIELD_CANONICAL[key] || "amount";
}

function getOptimizerFieldDescription(field) {
	const normalized = normalizeOptimizerField(field);
	return OPTIMIZER_FIELD_DESCRIPTIONS[normalized] || OPTIMIZER_FIELD_DESCRIPTIONS.amount;
}

function coalesceMonthValue(value, fallback) {
	if (typeof value === "string" && MONTH_PATTERN.test(value.trim())) {
		return value.trim();
	}
	return fallback;
}

function getDefaultEventStartMonth(event) {
	const candidates = [event?.startDate, currentConfig?.startDate, getCurrentMonthValue()];
	for (const candidate of candidates) {
		if (typeof candidate === "string" && MONTH_PATTERN.test(candidate.trim())) {
			return candidate.trim();
		}
	}
	return getCurrentMonthValue();
}

function getDefaultEventEndMonth(event) {
	const candidates = [event?.endDate, event?.startDate, currentConfig?.common?.deathDate, currentConfig?.startDate, getCurrentMonthValue()];
	for (const candidate of candidates) {
		if (typeof candidate === "string" && MONTH_PATTERN.test(candidate.trim())) {
			return candidate.trim();
		}
	}
	return getCurrentMonthValue();
}

function ensureOptimizerDefaults(event, field, options = {}) {
	const normalized = normalizeOptimizerField(field);
	if (!event.optimize || typeof event.optimize !== "object") {
		event.optimize = {};
	}
	const optimizer = event.optimize;
	optimizer.field = normalized;

	const resetBounds = Boolean(options.resetBounds);
 	const resetTolerance = Boolean(options.resetTolerance) || resetBounds;

	if (!Number.isFinite(optimizer.maxIterations) || optimizer.maxIterations <= 0) {
		optimizer.maxIterations = 50;
	}

	const ensureTolerance = (defaultValue) => {
		if (resetTolerance || !Number.isFinite(optimizer.tolerance) || optimizer.tolerance <= 0) {
			optimizer.tolerance = defaultValue;
		}
	};

	if (normalized === "amount") {
		delete optimizer.minDate;
		delete optimizer.maxDate;
		const defaultAmount = typeof event.amount === "number" && Number.isFinite(event.amount) ? event.amount : 0;
		if (resetBounds || !Number.isFinite(optimizer.min)) {
			optimizer.min = defaultAmount;
		}
		if (resetBounds || !Number.isFinite(optimizer.max)) {
			optimizer.max = defaultAmount;
		}
		ensureTolerance(0.01);
	} else if (normalized === "frequency") {
		delete optimizer.minDate;
		delete optimizer.maxDate;
		const defaultFrequency = typeof event.frequency === "number" && Number.isFinite(event.frequency) && event.frequency > 0
			? event.frequency
			: 1;
		if (resetBounds || !Number.isFinite(optimizer.min)) {
			optimizer.min = defaultFrequency;
		}
		if (resetBounds || !Number.isFinite(optimizer.max)) {
			optimizer.max = defaultFrequency;
		}
		ensureTolerance(1);
	} else if (normalized === "startDate") {
		delete optimizer.min;
		delete optimizer.max;
		const defaultMin = getDefaultEventStartMonth(event);
		const defaultMax = getDefaultEventEndMonth(event);
		const currentMin = typeof optimizer.minDate === "string" ? optimizer.minDate : "";
		const currentMax = typeof optimizer.maxDate === "string" ? optimizer.maxDate : "";
		if (resetBounds || !MONTH_PATTERN.test(currentMin.trim())) {
			optimizer.minDate = coalesceMonthValue(currentMin, defaultMin);
		}
		if (resetBounds || !MONTH_PATTERN.test(currentMax.trim())) {
			optimizer.maxDate = coalesceMonthValue(currentMax, defaultMax);
		}
		ensureTolerance(1);
	} else if (normalized === "endDate") {
		delete optimizer.min;
		delete optimizer.max;
		const defaultMin = getDefaultEventStartMonth(event);
		const defaultMax = getDefaultEventEndMonth(event);
		const currentMin = typeof optimizer.minDate === "string" ? optimizer.minDate : "";
		const currentMax = typeof optimizer.maxDate === "string" ? optimizer.maxDate : "";
		if (resetBounds || !MONTH_PATTERN.test(currentMin.trim())) {
			optimizer.minDate = coalesceMonthValue(currentMin, defaultMin);
		}
		if (resetBounds || !MONTH_PATTERN.test(currentMax.trim())) {
			optimizer.maxDate = coalesceMonthValue(currentMax, defaultMax);
		}
		ensureTolerance(1);
	} else {
		ensureTolerance(0.01);
	}

	return optimizer;
}

function getOptimizerFieldMeta(field) {
	const normalized = normalizeOptimizerField(field);
	if (normalized === "frequency") {
		return {
			minPath: "min",
			maxPath: "max",
			minLabel: "Min frequency (months)",
			maxLabel: "Max frequency (months)",
			minTooltip: "Smallest frequency, in months, the optimizer will consider.",
			maxTooltip: "Largest frequency, in months, the optimizer will consider.",
			inputType: "number",
			numberKind: "int",
			step: "1",
			arrowStep: ARROW_STEP_SMALL,
			validation: { type: "integer", min: 1, required: true },
			toleranceLabel: "Tolerance (months)",
			toleranceTooltip: "Stop when bounds differ by this many months. Leave blank or zero to use the default (1).",
			toleranceInputType: "number",
			toleranceNumberKind: "int",
			toleranceStep: "1",
			toleranceArrowStep: ARROW_STEP_SMALL,
			toleranceValidation: { type: "integer", min: 0 },
		};
	}
	if (normalized === "startDate") {
		return {
			minPath: "minDate",
			maxPath: "maxDate",
			minLabel: "Earliest start date",
			maxLabel: "Latest start date",
			minTooltip: "Earliest month the optimizer can consider for the event start date.",
			maxTooltip: "Latest month the optimizer can consider for the event start date.",
			inputType: "month",
			validation: { type: "month", required: true },
			maxLength: 7,
			enableNowShortcut: true,
			toleranceLabel: "Tolerance (months)",
			toleranceTooltip: "Stop when bounds differ by this many months. Leave blank or zero to use the default (1).",
			toleranceInputType: "number",
			toleranceNumberKind: "int",
			toleranceStep: "1",
			toleranceArrowStep: ARROW_STEP_SMALL,
			toleranceValidation: { type: "integer", min: 0 },
		};
	}
	if (normalized === "endDate") {
		return {
			minPath: "minDate",
			maxPath: "maxDate",
			minLabel: "Earliest end date",
			maxLabel: "Latest end date",
			minTooltip: "Earliest month the optimizer can consider for the event end date.",
			maxTooltip: "Latest month the optimizer can consider for the event end date.",
			inputType: "month",
			validation: { type: "month", required: true },
			maxLength: 7,
			enableNowShortcut: true,
			toleranceLabel: "Tolerance (months)",
			toleranceTooltip: "Stop when bounds differ by this many months. Leave blank or zero to use the default (1).",
			toleranceInputType: "number",
			toleranceNumberKind: "int",
			toleranceStep: "1",
			toleranceArrowStep: ARROW_STEP_SMALL,
			toleranceValidation: { type: "integer", min: 0 },
		};
	}
	return {
		minPath: "min",
		maxPath: "max",
		minLabel: "Min amount",
		maxLabel: "Max amount",
		minTooltip: "Smallest amount the optimizer will consider during the search.",
		maxTooltip: "Largest amount the optimizer will consider during the search.",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_LARGE,
		validation: { type: "number", required: true },
		toleranceLabel: "Tolerance",
		toleranceTooltip: "Stop when min and max differ by this amount. Leave blank or zero to use the default (0.01).",
		toleranceInputType: "number",
		toleranceStep: "0.01",
		toleranceArrowStep: ARROW_STEP_SMALL,
		toleranceValidation: { type: "number", min: 0 },
	};
}

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
window.addEventListener("resize", scheduleChartRerender);

initializeOptimizerControls();
initializeWorkspace();
initializeThemeControls();
initializeVersionFooter();

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

function isEditorStorageAvailable() {
	if (editorStorageAvailable !== null) {
		return editorStorageAvailable;
	}
	if (typeof window === "undefined" || !window.localStorage) {
		editorStorageAvailable = false;
		return editorStorageAvailable;
	}
	try {
		const testKey = "__ff_editor_storage_test__";
		window.localStorage.setItem(testKey, "1");
		window.localStorage.removeItem(testKey);
		editorStorageAvailable = true;
	} catch (error) {
		editorStorageAvailable = false;
		console.warn("Local storage is unavailable; editor drafts will not be persisted.", error);
	}
	return editorStorageAvailable;
}

function isQuotaExceededError(error) {
	if (!error) {
		return false;
	}
	return error.name === "QuotaExceededError"
		|| error.name === "NS_ERROR_DOM_QUOTA_REACHED"
		|| error.code === 22
		|| error.code === 1014;
}

function persistEditorState() {
	if (!currentConfig || !isEditorStorageAvailable()) {
		editorPersistTimer = null;
		return;
	}

	const snapshot = {
		version: EDITOR_STORAGE_VERSION,
		updatedAt: new Date().toISOString(),
		config: currentConfig,
		logging: hiddenLogging,
	};

	try {
		window.localStorage.setItem(EDITOR_STORAGE_KEY, JSON.stringify(snapshot));
	} catch (error) {
		if (isQuotaExceededError(error)) {
			console.warn("Unable to persist editor state: storage quota exceeded.");
		} else {
			console.warn("Unable to persist editor state.", error);
		}
	}

	editorPersistTimer = null;
}

function queuePersistEditorState(options = {}) {
	const { immediate = false } = options;
	if (!currentConfig || !isEditorStorageAvailable() || typeof window === "undefined") {
		return;
	}

	if (immediate) {
		if (editorPersistTimer !== null) {
			window.clearTimeout(editorPersistTimer);
			editorPersistTimer = null;
		}
		persistEditorState();
		return;
	}

	if (editorPersistTimer !== null) {
		window.clearTimeout(editorPersistTimer);
	}

	editorPersistTimer = window.setTimeout(() => {
		persistEditorState();
	}, EDITOR_PERSIST_DEBOUNCE_MS);
}

function clearPersistedEditorState() {
	if (!isEditorStorageAvailable() || typeof window === "undefined") {
		return;
	}

	if (editorPersistTimer !== null) {
		window.clearTimeout(editorPersistTimer);
		editorPersistTimer = null;
	}

	try {
		window.localStorage.removeItem(EDITOR_STORAGE_KEY);
	} catch (error) {
		console.warn("Unable to clear persisted editor state.", error);
	}
}

function loadPersistedEditorState() {
	if (!isEditorStorageAvailable() || typeof window === "undefined") {
		return null;
	}

	let rawValue = null;
	try {
		rawValue = window.localStorage.getItem(EDITOR_STORAGE_KEY);
	} catch (error) {
		console.warn("Unable to access persisted editor state.", error);
		return null;
	}

	if (!rawValue) {
		return null;
	}

	try {
		const parsed = JSON.parse(rawValue);
		if (!parsed || parsed.version !== EDITOR_STORAGE_VERSION || !parsed.config) {
			return null;
		}
		const prepared = prepareConfigForEditing(parsed.config);
		const logging = parsed.logging ? cloneDeep(parsed.logging) : prepared.logging || null;
		return {
			config: prepared.config,
			logging: logging || getDefaultLoggingConfig(),
			updatedAt: parsed.updatedAt || null,
		};
	} catch (error) {
		console.warn("Failed to parse persisted editor state. Clearing saved draft.", error);
		try {
			window.localStorage.removeItem(EDITOR_STORAGE_KEY);
		} catch (removeError) {
			console.warn("Unable to clear invalid persisted editor state.", removeError);
		}
		return null;
	}
}

function setupEditorPersistenceHandlers() {
	if (editorPersistenceHandlersRegistered || typeof window === "undefined") {
		return;
	}
	if (!isEditorStorageAvailable()) {
		return;
	}

	const flush = () => queuePersistEditorState({ immediate: true });
	window.addEventListener("beforeunload", flush);
	window.addEventListener("pagehide", flush);
	editorPersistenceHandlersRegistered = true;
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

		setOptimizerEnabledState(false, { skipRender: true });
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
	const rows = Array.isArray(data?.rows) ? [...data.rows] : [];
	const metrics = Array.isArray(data?.metrics) ? data.metrics : [];
	forecastDataset = { scenarios, rows, metrics };
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
	scrollResultsToTop({ behavior: "auto" });
	showMessage(successMessage, "success");
}

function showMessage(message, type) {
	const trimmedMessage = typeof message === "string" ? message.trim() : "";
	if (trimmedMessage === "") {
		messageEl.textContent = "";
		messageEl.className = "message hidden";
		clearStickyInlineError();
		return;
	}

	messageEl.textContent = trimmedMessage;
	const classes = ["message"];
	if (type) {
		classes.push(type);
	}
	messageEl.className = classes.join(" ");

	if (type === "error") {
		if (activeTab === "config") {
			setStickyInlineError(trimmedMessage);
		} else {
			clearStickyInlineError();
		}
	} else {
		clearStickyInlineError();
	}
}

function clearStickyInlineError() {
	if (stickyInlineErrorEl && stickyInlineErrorEl.parentNode) {
		stickyInlineErrorEl.parentNode.removeChild(stickyInlineErrorEl);
	}
	stickyInlineErrorEl = null;
	stickyInlineErrorAnchor = null;
}

function findActiveStickyAnchor() {
	if (!configPanel) {
		return null;
	}
	const selectors = [".sticky-heading", ".scenario-card-header"];
	const nodes = configPanel.querySelectorAll(selectors.join(", "));
	if (nodes.length === 0) {
		return null;
	}
	const offset = getNumericCSSValue("--workspace-sticky-offset") + getNumericCSSValue("--config-toolbar-offset");
	const threshold = 6;
	let fallback = null;
	for (const node of nodes) {
		if (!node.isConnected) {
			continue;
		}
		const rect = node.getBoundingClientRect();
		if (Number.isNaN(rect.top) || Number.isNaN(rect.bottom)) {
			continue;
		}
		if (Math.abs(rect.top - offset) <= threshold) {
			return node;
		}
		if (!fallback && rect.bottom >= 0) {
			fallback = node;
		}
	}
	return fallback || nodes[0];
}

function findFallbackStickyAnchor() {
	if (!configPanel) {
		return null;
	}
	return configPanel.querySelector(".scenario-card-header, .sticky-heading");
}

function setStickyInlineError(message) {
	const trimmed = typeof message === "string" ? message.trim() : "";
	if (trimmed === "") {
		clearStickyInlineError();
		return;
	}
	let anchor = stickyInlineErrorAnchor && stickyInlineErrorAnchor.isConnected ? stickyInlineErrorAnchor : null;
	if (!anchor) {
		anchor = findActiveStickyAnchor() || findFallbackStickyAnchor();
	}
	if (!anchor) {
		clearStickyInlineError();
		return;
	}
	if (stickyInlineErrorAnchor !== anchor) {
		clearStickyInlineError();
		const isHeading = anchor.classList.contains("sticky-heading");
		const elementTag = isHeading ? "span" : "div";
		stickyInlineErrorEl = document.createElement(elementTag);
		stickyInlineErrorEl.className = "sticky-inline-alert sticky-inline-alert--error";
		stickyInlineErrorEl.setAttribute("role", "alert");
		stickyInlineErrorEl.setAttribute("aria-live", "assertive");
		if (isHeading) {
			anchor.appendChild(stickyInlineErrorEl);
		} else if (anchor.classList.contains("scenario-card-header")) {
			const titleEl = anchor.querySelector("h4");
			if (titleEl) {
				titleEl.insertAdjacentElement("afterend", stickyInlineErrorEl);
			} else {
				anchor.appendChild(stickyInlineErrorEl);
			}
		} else {
			anchor.appendChild(stickyInlineErrorEl);
		}
		stickyInlineErrorAnchor = anchor;
	}
	if (stickyInlineErrorEl) {
		stickyInlineErrorEl.textContent = trimmed;
	}
}

function getNumericCSSValue(variableName) {
	if (typeof window === "undefined" || !variableName) {
		return 0;
	}
	const computed = window.getComputedStyle(document.documentElement);
	const rawValue = computed.getPropertyValue(variableName);
	const parsed = parseFloat(rawValue);
	return Number.isFinite(parsed) ? parsed : 0;
}

function scrollResultsToTop(options = {}) {
	if (!resultsPanel) {
		return;
	}

	const { behavior = "auto" } = options || {};
	window.requestAnimationFrame(() => {
		const rect = resultsPanel.getBoundingClientRect();
		const stickyOffset = getNumericCSSValue("--workspace-sticky-offset");
		const targetTop = rect.top + window.scrollY - stickyOffset - 16;
		const clampedTop = Math.max(0, targetTop);
		window.scrollTo({
			top: clampedTop,
			behavior: behavior === "smooth" ? "smooth" : "auto",
		});
	});
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

	if (chartWrapper) {
		chartWrapper.classList.add("hidden");
	}
	if (chartLegendEl) {
		chartLegendEl.innerHTML = "";
		chartLegendEl.classList.add("hidden");
	}
	if (chartSvg) {
		while (chartSvg.firstChild) {
			chartSvg.removeChild(chartSvg.firstChild);
		}
		chartSvg.classList.add("hidden");
	}
	if (chartEmptyEl) {
		chartEmptyEl.classList.add("hidden");
	}
	if (chartCaptionEl) {
		chartCaptionEl.textContent = "Line chart showing liquid and total net worth over time for the selected scenario.";
	}
	if (resultsSummaryEl) {
		resultsSummaryEl.textContent = "";
		resultsSummaryEl.classList.add("hidden");
	}
	if (chartResizeFrame !== null) {
		window.cancelAnimationFrame(chartResizeFrame);
		chartResizeFrame = null;
	}

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
	renderScenarioSummary();
	renderScenarioChart();
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



function renderScenarioSummary() {
	if (!resultsSummaryEl) {
		return;
	}

	resultsSummaryEl.innerHTML = "";
	resultsSummaryEl.classList.add("hidden");

	if (!forecastDataset || !Array.isArray(forecastDataset.scenarios) || forecastDataset.scenarios.length === 0) {
		return;
	}

	const scenarioIndex = clampActiveScenarioIndex();
	const metrics = Array.isArray(forecastDataset.metrics) ? forecastDataset.metrics[scenarioIndex] : null;
	const rows = Array.isArray(forecastDataset.rows) ? forecastDataset.rows : [];

	let hasContent = false;

	const negativeSpans = findNegativeNetWorthSpans(rows, scenarioIndex);
	if (negativeSpans.length > 0) {
		const warningBlock = document.createElement("div");
		warningBlock.className = "results-summary__warning";
		const heading = document.createElement("div");
		heading.className = "results-summary__heading results-summary__heading--warning";
		heading.textContent = "Warning: Net worth below $0";
		warningBlock.appendChild(heading);

		const list = document.createElement("ul");
		list.className = "results-summary__list results-summary__list--warning";

		negativeSpans.forEach((span) => {
			const item = document.createElement("li");
			item.className = "results-summary__item";
			const minDisplay = formatSummaryCurrency(span.minValue) || String(span.minValue);
			const metricLabel = getNetWorthMetricLabel(span.metric);
			const entryText = span.length === 1
				? `${metricLabel} dips below zero in ${span.startDate} (low ${minDisplay})`
				: `${metricLabel} below zero from ${span.startDate} to ${span.endDate} (${span.length} months, low ${minDisplay})`;
			item.textContent = entryText;
			list.appendChild(item);
		});

		warningBlock.appendChild(list);
		resultsSummaryEl.appendChild(warningBlock);
		hasContent = true;
	}

	if (!metrics) {
		if (hasContent) {
			resultsSummaryEl.classList.remove("hidden");
		}
		return;
	}

	if (metrics.emergencyFund) {
		const ef = metrics.emergencyFund;
		const parts = [];
		if (typeof ef.targetMonths === "number" && Number.isFinite(ef.targetMonths)) {
			parts.push(`Emergency fund target (${ef.targetMonths.toFixed(1)} months)`);
		}
		if (typeof ef.targetAmount === "number" && Number.isFinite(ef.targetAmount)) {
			parts.push(`Goal: ${SUMMARY_CURRENCY_FORMATTER.format(ef.targetAmount)}`);
		}
		if (typeof ef.averageMonthlyExpenses === "number" && ef.averageMonthlyExpenses > 0) {
			parts.push(`Avg expenses: ${SUMMARY_CURRENCY_FORMATTER.format(ef.averageMonthlyExpenses)}`);
		}
		if (typeof ef.fundedMonths === "number" && Number.isFinite(ef.fundedMonths)) {
			parts.push(`Starting coverage: ${ef.fundedMonths.toFixed(1)} months`);
		}
		if (typeof ef.shortfall === "number" && ef.shortfall > 0) {
			parts.push(`Shortfall: ${SUMMARY_CURRENCY_FORMATTER.format(ef.shortfall)}`);
		} else if (typeof ef.surplus === "number" && ef.surplus > 0) {
			parts.push(`Surplus: ${SUMMARY_CURRENCY_FORMATTER.format(ef.surplus)}`);
		}

		if (parts.length > 0) {
			const efBlock = document.createElement("div");
			efBlock.className = "results-summary__emergency";
			const heading = document.createElement("div");
			heading.className = "results-summary__heading";
			heading.textContent = "Emergency fund summary";
			efBlock.appendChild(heading);

			const list = document.createElement("ul");
			list.className = "results-summary__list";
			const item = document.createElement("li");
			const description = document.createElement("div");
			description.className = "results-summary__item";
			description.textContent = parts.join(" • ");
			item.appendChild(description);
			list.appendChild(item);
			efBlock.appendChild(list);
			resultsSummaryEl.appendChild(efBlock);
			hasContent = true;
		}
	}

	const optimizations = Array.isArray(metrics.optimizations) ? metrics.optimizations : [];
	if (optimizations.length > 0) {
		const convergedSummaries = optimizations.filter((summary) => summary && summary.converged);
		const incompleteSummaries = optimizations.filter((summary) => summary && !summary.converged);

		if (convergedSummaries.length > 0) {
			const optimizerBlock = document.createElement("div");
			optimizerBlock.className = "results-summary__optimizer";
			const heading = document.createElement("div");
			heading.className = "results-summary__heading";
			heading.textContent = convergedSummaries.length === 1 ? "Optimizer adjustment" : "Optimizer adjustments";
			optimizerBlock.appendChild(heading);

			const list = document.createElement("ul");
			list.className = "results-summary__list";

			convergedSummaries.forEach((summary) => {
				const item = document.createElement("li");
				const description = document.createElement("div");
				description.className = "results-summary__item";
				const eventLabel = typeof summary.targetName === "string" && summary.targetName.trim() !== "" ? summary.targetName.trim() : "Event";
				const fieldLabel = typeof summary.field === "string" && summary.field.trim() !== "" ? summary.field.trim() : "amount";
				const label = fieldLabel.toLowerCase() === "amount" ? eventLabel : `${eventLabel} (${fieldLabel})`;
				const originalValue = formatOptimizerDisplay(summary, "original");
				const optimizedValue = formatOptimizerDisplay(summary, "value");
				const parts = [label];
				if (originalValue && optimizedValue) {
					parts.push(`${originalValue} → ${optimizedValue}`);
				}
				description.textContent = parts.join(" • ");
				item.appendChild(description);

				const detailParts = [];
				if (typeof summary.floor === "number" && Number.isFinite(summary.floor)) {
					detailParts.push(`Cash floor: ${formatSummaryCurrency(summary.floor)}`);
				}
				if (typeof summary.minimumCash === "number" && Number.isFinite(summary.minimumCash)) {
					detailParts.push(`Minimum cash: ${formatSummaryCurrency(summary.minimumCash)}`);
				}
				if (typeof summary.headroom === "number" && Number.isFinite(summary.headroom)) {
					detailParts.push(`Headroom: ${formatSummaryCurrency(summary.headroom)}`);
				}
				if (typeof summary.iterations === "number" && summary.iterations > 0) {
					detailParts.push(`Iterations: ${summary.iterations}`);
				}
				if (detailParts.length > 0) {
					const details = document.createElement("div");
					details.className = "results-summary__notes muted-text";
					details.textContent = detailParts.join(" • ");
					item.appendChild(details);
				}

				list.appendChild(item);
			});

			optimizerBlock.appendChild(list);
			resultsSummaryEl.appendChild(optimizerBlock);
			hasContent = true;
		}

		if (incompleteSummaries.length > 0) {
			const warningBlock = document.createElement("div");
			warningBlock.className = "results-summary__optimizer";
			const heading = document.createElement("div");
			heading.className = "results-summary__heading";
			heading.textContent = "Optimizer notes";
			warningBlock.appendChild(heading);

			const list = document.createElement("ul");
			list.className = "results-summary__list";

			incompleteSummaries.forEach((summary) => {
				const item = document.createElement("li");
				const description = document.createElement("div");
				description.className = "results-summary__item";
				const eventLabel = typeof summary.targetName === "string" && summary.targetName.trim() !== "" ? summary.targetName.trim() : "Event";
				const fieldLabel = typeof summary.field === "string" && summary.field.trim() !== "" ? summary.field.trim() : "amount";
				const label = fieldLabel.toLowerCase() === "amount" ? eventLabel : `${eventLabel} (${fieldLabel})`;
				const originalValue = formatOptimizerDisplay(summary, "original");
				const attemptedValue = formatOptimizerDisplay(summary, "value");
				const parts = [label];
				if (originalValue && attemptedValue) {
					parts.push(`${originalValue} → ${attemptedValue}`);
				}
				description.textContent = parts.join(" • ");
				item.appendChild(description);

				const notes = Array.isArray(summary.notes)
					? summary.notes
						.map((note) => (typeof note === "string" ? note.trim() : ""))
						.filter((note) => note !== "")
					: [];
				const notesEl = document.createElement("div");
				notesEl.className = "results-summary__notes muted-text";
				if (notes.length > 0) {
					notesEl.textContent = notes.join(" • ");
				} else {
					notesEl.textContent = "Unable to reach the emergency fund floor within the configured bounds.";
				}
				item.appendChild(notesEl);
				list.appendChild(item);
			});

			warningBlock.appendChild(list);
			resultsSummaryEl.appendChild(warningBlock);
			hasContent = true;
		}
	}

	if (hasContent) {
		resultsSummaryEl.classList.remove("hidden");
	}
}

function renderScenarioChart() {
	if (!chartWrapper || !chartSvg || !chartLegendEl) {
		return;
	}

	if (chartTooltipEl) {
		chartTooltipEl.classList.add("hidden");
		chartTooltipEl.setAttribute("aria-hidden", "true");
	}

	while (chartSvg.firstChild) {
		chartSvg.removeChild(chartSvg.firstChild);
	}
	chartSvg.classList.add("hidden");
	chartLegendEl.innerHTML = "";
	chartLegendEl.classList.add("hidden");
	if (chartEmptyEl) {
		chartEmptyEl.classList.add("hidden");
	}

	if (!forecastDataset || !Array.isArray(forecastDataset.scenarios) || forecastDataset.scenarios.length === 0) {
		if (chartWrapper) {
			chartWrapper.classList.add("hidden");
		}
		return;
	}

	const scenarioIndex = clampActiveScenarioIndex();
	const scenarioName = forecastDataset.scenarios[scenarioIndex] || `Scenario ${scenarioIndex + 1}`;
	const rows = Array.isArray(forecastDataset.rows) ? forecastDataset.rows : [];

	const points = rows
		.map((row) => {
			const parsedDate = parseForecastDate(row.date);
			if (!parsedDate) {
				return null;
			}
			const value = Array.isArray(row.values) ? row.values[scenarioIndex] || null : null;
			const liquid = getScenarioValue(value, "liquid");
			const total = getScenarioValue(value, "total");
			if (liquid === null && total === null) {
				return null;
			}
			return {
				dateLabel: row.date,
				date: parsedDate,
				time: parsedDate.getTime(),
				liquid,
				total,
			};
		})
		.filter(Boolean);

	if (chartTitleEl) {
		chartTitleEl.textContent = `Net Worth Over Time — ${scenarioName}`;
	}
	if (chartCaptionEl) {
		chartCaptionEl.textContent = `Line chart showing liquid and total net worth over time for the selected scenario: ${scenarioName}.`;
	}
	chartSvg.setAttribute("aria-label", `Line chart of liquid and total net worth for ${scenarioName}.`);

	buildChartLegend(points);

	if (points.length === 0) {
		chartWrapper.classList.remove("hidden");
		chartLegendEl.classList.remove("hidden");
		if (chartEmptyEl) {
			chartEmptyEl.classList.remove("hidden");
		}
		updateStickyMetrics();
		return;
	}

	const sortedPoints = points.slice().sort((a, b) => a.time - b.time);
	const containerWidth = Math.max(chartWrapper.clientWidth || 0, 480);
	const width = containerWidth;
	const height = Math.max(
		CHART_MIN_HEIGHT,
		Math.min(CHART_MAX_HEIGHT, Math.round(containerWidth * CHART_ASPECT_RATIO)),
	);

	chartSvg.setAttribute("viewBox", `0 0 ${width} ${height}`);
	chartSvg.setAttribute("width", width);
	chartSvg.setAttribute("height", height);
	chartSvg.setAttribute("preserveAspectRatio", "xMidYMid meet");
	chartSvg.classList.remove("hidden");
	chartWrapper.classList.remove("hidden");
	chartLegendEl.classList.remove("hidden");
	if (chartEmptyEl) {
		chartEmptyEl.classList.add("hidden");
	}

	const existingDesc = chartSvg.querySelector("desc");
	if (existingDesc) {
		chartSvg.removeChild(existingDesc);
	}
	const descEl = createSvgElement("desc");
	descEl.textContent = `Liquid and total net worth for ${scenarioName} from ${sortedPoints[0].dateLabel} to ${sortedPoints[sortedPoints.length - 1].dateLabel}.`;
	chartSvg.insertBefore(descEl, chartSvg.firstChild);

	const xValues = sortedPoints.map((point) => point.time);
	const yValues = [];
	sortedPoints.forEach((point) => {
		if (typeof point.liquid === "number" && Number.isFinite(point.liquid)) {
			yValues.push(point.liquid);
		}
		if (typeof point.total === "number" && Number.isFinite(point.total)) {
			yValues.push(point.total);
		}
	});

	if (yValues.length > 0) {
		const min = Math.min(...yValues);
		const max = Math.max(...yValues);
		if (min > 0) {
			yValues.push(0);
		}
		if (max < 0) {
			yValues.push(0);
		}
	} else {
		yValues.push(0);
	}

	let xMin = Math.min(...xValues);
	let xMax = Math.max(...xValues);
	if (!Number.isFinite(xMin) || !Number.isFinite(xMax)) {
		xMin = Date.now();
		xMax = xMin + 1;
	}
	if (xMin === xMax) {
		const halfWindow = 1000 * 60 * 60 * 24 * 15;
		xMin -= halfWindow;
		xMax += halfWindow;
	}

	let yMin = Math.min(...yValues);
	let yMax = Math.max(...yValues);
	if (!Number.isFinite(yMin) || !Number.isFinite(yMax)) {
		yMin = 0;
		yMax = 1;
	}
	if (yMin === yMax) {
		const pad = Math.max(Math.abs(yMin) * 0.15, 1000);
		yMin -= pad;
		yMax += pad;
	} else {
		const pad = (yMax - yMin) * 0.08;
		yMin -= pad;
		yMax += pad;
	}

	const plotLeftX = CHART_MARGIN.left;
	const plotRightX = width - CHART_MARGIN.right;
	const plotTopY = CHART_MARGIN.top;
	const plotBottomY = height - CHART_MARGIN.bottom;
	const plotWidth = Math.max(0, plotRightX - plotLeftX);
	const plotHeight = Math.max(0, plotBottomY - plotTopY);

	const xScale = createLinearScale(xMin, xMax, plotLeftX, plotRightX);
	const yScale = createLinearScale(yMin, yMax, plotBottomY, plotTopY);

	const bandsGroup = createSvgElement("g", { class: "chart-bands" });
	const gridGroup = createSvgElement("g", { class: "chart-grid" });
	const axesGroup = createSvgElement("g", { class: "chart-axes" });
	const linesGroup = createSvgElement("g", { class: "chart-lines" });
	const pointsGroup = createSvgElement("g", { class: "chart-points" });
	const interactionGroup = createSvgElement("g", { class: "chart-interaction" });
	chartSvg.appendChild(bandsGroup);
	chartSvg.appendChild(gridGroup);
	chartSvg.appendChild(linesGroup);
	chartSvg.appendChild(pointsGroup);
	chartSvg.appendChild(axesGroup);
	chartSvg.appendChild(interactionGroup);

	const currencyFormatter = new Intl.NumberFormat(undefined, {
		style: "currency",
		currency: "USD",
		maximumFractionDigits: 0,
		notation: "compact",
		compactDisplay: "short",
	});
	const dateFormatter = new Intl.DateTimeFormat(undefined, {
		month: "short",
		year: "numeric",
	});

	const zeroY = yMin <= 0 && yMax >= 0 ? yScale(0) : null;
	if (yMin < 0 && Number.isFinite(zeroY) && plotHeight > 0) {
		const bandY = Math.min(zeroY, plotBottomY);
		const bandHeight = Math.abs(plotBottomY - zeroY);
		if (bandHeight > 0.5) {
			const negativeBand = createSvgElement("rect", {
				class: "chart-negative-band",
				x: plotLeftX,
				y: bandY,
				width: plotWidth,
				height: bandHeight,
			});
			bandsGroup.appendChild(negativeBand);
		}
	}

	const negativeSegments = calculateNegativeNetWorthSegments(sortedPoints);
	negativeSegments.forEach(({ startIndex, endIndex, metric }) => {
		if (typeof startIndex !== "number" || typeof endIndex !== "number" || startIndex < 0 || endIndex < startIndex) {
			return;
		}
		if (startIndex === endIndex) {
			const point = sortedPoints[startIndex];
			const x = point ? xScale(point.time) : NaN;
			if (Number.isFinite(x)) {
				const markerClasses = ["chart-negative-marker"];
				if (metric) {
					markerClasses.push(`chart-negative-marker--${metric}`);
				}
				const markerAttributes = {
					class: markerClasses.join(" "),
					x1: x,
					x2: x,
					y1: plotTopY,
					y2: plotBottomY,
				};
				if (metric) {
					markerAttributes["data-metric"] = metric;
				}
				const marker = createSvgElement("line", markerAttributes);
				bandsGroup.appendChild(marker);
			}
			return;
		}
		const bounds = calculateSegmentDomainBounds(sortedPoints, startIndex, endIndex);
		if (!bounds) {
			return;
		}
		const xStart = xScale(bounds.startTime);
		const xEnd = xScale(bounds.endTime);
		if (!Number.isFinite(xStart) || !Number.isFinite(xEnd)) {
			return;
		}
		const width = Math.abs(xEnd - xStart);
		if (width < 1) {
			const midPoint = sortedPoints[startIndex];
			const midX = midPoint ? xScale(midPoint.time) : NaN;
			if (Number.isFinite(midX)) {
				const marker = createSvgElement("line", {
					class: "chart-negative-marker",
					x1: midX,
					x2: midX,
					y1: plotTopY,
					y2: plotBottomY,
				});
				bandsGroup.appendChild(marker);
			}
			return;
		}
		const rectClasses = ["chart-negative-period"];
		if (metric) {
			rectClasses.push(`chart-negative-period--${metric}`);
		}
		const rectAttributes = {
			class: rectClasses.join(" "),
			x: Math.min(xStart, xEnd),
			y: plotTopY,
			width,
			height: plotHeight,
		};
		if (metric) {
			rectAttributes["data-metric"] = metric;
		}
		const rect = createSvgElement("rect", rectAttributes);
		bandsGroup.appendChild(rect);
	});

	const yTicks = generateLinearTicks(yMin, yMax, 5);
	yTicks.forEach((tick) => {
		if (!Number.isFinite(tick)) {
			return;
		}
		const y = yScale(tick);
		if (!Number.isFinite(y)) {
			return;
		}
		if (y < plotTopY - 0.5 || y > plotBottomY + 0.5) {
			return;
		}
		const gridLine = createSvgElement("line", {
			class: "chart-grid-line",
			x1: plotLeftX,
			x2: plotRightX,
			y1: y,
			y2: y,
		});
		gridGroup.appendChild(gridLine);

		const label = createSvgElement("text", {
			class: "chart-axis-label",
			x: plotLeftX - 18,
			y,
			"text-anchor": "end",
			"dominant-baseline": "middle",
		});
		label.textContent = currencyFormatter.format(tick);
		axesGroup.appendChild(label);
	});

	const xTickCount = Math.max(3, Math.min(8, Math.round(width / 160)));
	const xTicks = generateTimeTicks(sortedPoints, xTickCount);
	xTicks.forEach((tickPoint) => {
		const x = xScale(tickPoint.time);
		if (!Number.isFinite(x)) {
			return;
		}
		if (x > plotLeftX + 0.5 && x < plotRightX - 0.5) {
			const verticalLine = createSvgElement("line", {
				class: "chart-grid-line",
				x1: x,
				x2: x,
				y1: plotTopY,
				y2: plotBottomY,
			});
			gridGroup.appendChild(verticalLine);
		}
		const label = createSvgElement("text", {
			class: "chart-axis-tick",
			x,
			y: plotBottomY + 16,
			"text-anchor": "middle",
		});
		label.textContent = dateFormatter.format(tickPoint.date);
		axesGroup.appendChild(label);
	});

	const yAxisLine = createSvgElement("line", {
		class: "chart-axis",
		x1: plotLeftX,
		x2: plotLeftX,
		y1: plotTopY,
		y2: plotBottomY,
	});
	const xAxisLine = createSvgElement("line", {
		class: "chart-axis",
		x1: plotLeftX,
		x2: plotRightX,
		y1: plotBottomY,
		y2: plotBottomY,
	});
	axesGroup.appendChild(yAxisLine);
	axesGroup.appendChild(xAxisLine);

	if (Number.isFinite(zeroY)) {
		const zeroLine = createSvgElement("line", {
			class: "chart-reference-line",
			x1: plotLeftX,
			x2: plotRightX,
			y1: zeroY,
			y2: zeroY,
		});
		gridGroup.appendChild(zeroLine);
	}
	const yAxisLabel = createSvgElement("text", {
		class: "chart-axis-label chart-axis-label-y",
		x: plotLeftX - 58,
		y: (plotTopY + plotBottomY) / 2,
	});
	yAxisLabel.textContent = "Net Worth (USD)";
	axesGroup.appendChild(yAxisLabel);

	const xAxisLabel = createSvgElement("text", {
		class: "chart-axis-label chart-axis-label-x",
		x: (plotLeftX + plotRightX) / 2,
		y: plotBottomY + 44,
	});
	xAxisLabel.textContent = "Time";
	axesGroup.appendChild(xAxisLabel);

	CHART_SERIES.forEach((series) => {
		const linePath = buildLinePath(sortedPoints, (point) => point[series.key], xScale, yScale);
		if (!linePath) {
			return;
		}
		const pathElement = createSvgElement("path", {
			class: `chart-line ${series.lineClass}`,
			d: linePath,
		});
		linesGroup.appendChild(pathElement);

		const seriesPoints = sortedPoints.filter((point) => typeof point[series.key] === "number" && Number.isFinite(point[series.key]));
		if (seriesPoints.length === 0) {
			return;
		}
		const lastPoint = seriesPoints[seriesPoints.length - 1];
		const circle = createSvgElement("circle", {
			class: `chart-point ${series.pointClass}`,
			cx: xScale(lastPoint.time),
			cy: yScale(lastPoint[series.key]),
			r: 4,
		});
		pointsGroup.appendChild(circle);
	});

	setupChartTooltip({
		sortedPoints,
		xScale,
		plotLeftX,
		plotRightX,
		plotTopY,
		plotBottomY,
		xMin,
		xMax,
		interactionGroup,
		chartSvg,
	});

	updateStickyMetrics();
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
		const hasNegative = (typeof liquidAmount === "number" && liquidAmount < 0)
			|| (typeof totalAmount === "number" && totalAmount < 0);
		if (hasNegative) {
			tr.classList.add("results-row--negative");
		}
		const liquidValue = liquidAmount !== null ? currencyFormatter.format(liquidAmount) : noValueMarkup;
		const totalValue = totalAmount !== null ? currencyFormatter.format(totalAmount) : noValueMarkup;

		tr.appendChild(createCell(liquidValue, "amount-cell"));
		tr.appendChild(createCell(totalValue, "amount-cell"));
		tr.appendChild(createCell(formatNotes(value.notes)));

		tableBody.appendChild(tr);
	});
}

	function buildChartLegend(points) {
		if (!chartLegendEl) {
			return;
		}

		chartLegendEl.innerHTML = "";
		const entries = Array.isArray(points) ? points : [];
		const hasAnyEntries = entries.length > 0;

		CHART_SERIES.forEach((series) => {
			const hasValues = hasAnyEntries
				? entries.some((point) => typeof point[series.key] === "number" && Number.isFinite(point[series.key]))
				: false;
			const item = document.createElement("span");
			item.className = "chart-legend-item";
			item.setAttribute("role", "listitem");
			if (!hasValues) {
				item.classList.add("chart-legend-item--muted");
				item.setAttribute("aria-disabled", "true");
			}
			const swatch = document.createElement("span");
			swatch.className = `chart-legend-swatch ${series.swatchClass}`;
			item.appendChild(swatch);
			const label = document.createElement("span");
			label.textContent = series.label;
			item.appendChild(label);
			chartLegendEl.appendChild(item);
		});

		if (chartLegendEl.children.length === 0) {
			chartLegendEl.classList.add("hidden");
		} else {
			chartLegendEl.classList.remove("hidden");
		}
	}

	function getScenarioValue(value, key) {
		if (!value || typeof value !== "object") {
			return null;
		}
		const candidate = value[key];
		if (typeof candidate === "number" && Number.isFinite(candidate)) {
			return candidate;
		}
		if (typeof value.amount === "number" && Number.isFinite(value.amount)) {
			return value.amount;
		}
		return null;
	}

	function parseForecastDate(raw) {
		if (typeof raw !== "string" || raw.trim() === "") {
			return null;
		}
		const normalized = raw.trim();
		if (MONTH_PATTERN.test(normalized)) {
			const [yearStr, monthStr] = normalized.split("-");
			const year = Number(yearStr);
			const month = Number(monthStr) - 1;
			if (Number.isFinite(year) && Number.isFinite(month)) {
				return new Date(year, month, 1);
			}
		}
		const parsed = Date.parse(normalized);
		if (!Number.isNaN(parsed)) {
			return new Date(parsed);
		}
		return null;
	}

	function createSvgElement(tag, attributes = {}) {
		const element = document.createElementNS(SVG_NS, tag);
		Object.entries(attributes).forEach(([name, value]) => {
			if (value === null || typeof value === "undefined" || (typeof value === "number" && Number.isNaN(value))) {
				return;
			}
			element.setAttribute(name, String(value));
		});
		return element;
	}

	function createLinearScale(domainMin, domainMax, rangeMin, rangeMax) {
		if (!Number.isFinite(domainMin) || !Number.isFinite(domainMax) || domainMin === domainMax) {
			const center = (rangeMin + rangeMax) / 2;
			return () => center;
		}
		const domainSpan = domainMax - domainMin;
		const rangeSpan = rangeMax - rangeMin;
		return (value) => {
			if (!Number.isFinite(value)) {
				return NaN;
			}
			const ratio = (value - domainMin) / domainSpan;
			return rangeMin + ratio * rangeSpan;
		};
	}

	function buildLinePath(points, accessor, xScale, yScale) {
		let path = "";
		let segmentOpen = false;
		points.forEach((point) => {
			const value = accessor(point);
			if (typeof value !== "number" || !Number.isFinite(value)) {
				segmentOpen = false;
				return;
			}
			const x = xScale(point.time);
			const y = yScale(value);
			if (!Number.isFinite(x) || !Number.isFinite(y)) {
				segmentOpen = false;
				return;
			}
			if (!segmentOpen) {
				path += `M${x} ${y}`;
				segmentOpen = true;
			} else {
				path += ` L${x} ${y}`;
			}
		});
		return path.length > 0 ? path : null;
	}

	function generateLinearTicks(min, max, count = 5) {
		if (!Number.isFinite(min) || !Number.isFinite(max)) {
			return [];
		}
		if (min === max) {
			return [min];
		}
		const span = max - min;
		const step = niceNumber(span / Math.max(1, count - 1), true);
		const niceMin = Math.floor(min / step) * step;
		const niceMax = Math.ceil(max / step) * step;
		const ticks = [];
		for (let tick = niceMin; tick <= niceMax + step * 0.5; tick += step) {
			ticks.push(Number(tick.toFixed(10)));
		}
		return ticks;
	}

	function niceNumber(value, round) {
		if (!Number.isFinite(value) || value === 0) {
			return 1;
		}
		const exponent = Math.floor(Math.log10(Math.abs(value)));
		const fraction = Math.abs(value) / 10 ** exponent;
		let niceFraction;

		if (round) {
			if (fraction < 1.5) {
				niceFraction = 1;
			} else if (fraction < 3) {
				niceFraction = 2;
			} else if (fraction < 7) {
				niceFraction = 5;
			} else {
				niceFraction = 10;
			}
		} else if (fraction <= 1) {
			niceFraction = 1;
		} else if (fraction <= 2) {
			niceFraction = 2;
		} else if (fraction <= 5) {
			niceFraction = 5;
		} else {
			niceFraction = 10;
		}

		return niceFraction * 10 ** exponent;
	}

	function generateTimeTicks(points, desiredCount) {
		if (!Array.isArray(points) || points.length === 0) {
			return [];
		}
		if (points.length <= desiredCount) {
			return points;
		}
		const step = Math.max(1, Math.ceil(points.length / desiredCount));
		const ticks = [];
		for (let index = 0; index < points.length; index += step) {
			ticks.push(points[index]);
		}
		const lastPoint = points[points.length - 1];
		if (ticks[ticks.length - 1]?.time !== lastPoint.time) {
			ticks.push(lastPoint);
		}
		return ticks;
	}

		function setupChartTooltip({
			sortedPoints,
			xScale,
			plotLeftX,
			plotRightX,
			plotTopY,
			plotBottomY,
			xMin,
			xMax,
			interactionGroup,
			chartSvg,
		}) {
			if (!chartTooltipEl || !chartWrapper || !interactionGroup) {
				return;
			}
			if (!Array.isArray(sortedPoints) || sortedPoints.length === 0) {
				return;
			}

			const plotWidth = Math.max(0, plotRightX - plotLeftX);
			const plotHeight = Math.max(0, plotBottomY - plotTopY);
			if (plotWidth <= 0 || plotHeight <= 0) {
				return;
			}

			const tooltipPoints = sortedPoints
				.map((point) => {
					const x = xScale(point.time);
					if (!Number.isFinite(x)) {
						return null;
					}
					return {
						data: point,
						x,
						time: point.time,
					};
				})
				.filter(Boolean);

			if (tooltipPoints.length === 0) {
				return;
			}

			const overlayRect = createSvgElement("rect", {
				class: "chart-overlay",
				x: plotLeftX,
				y: plotTopY,
				width: plotWidth,
				height: plotHeight,
				fill: "transparent",
				"pointer-events": "all",
			});
			const pointerLine = createSvgElement("line", {
				class: "chart-pointer-line",
				x1: plotLeftX,
				x2: plotLeftX,
				y1: plotTopY,
				y2: plotBottomY,
				"pointer-events": "none",
			});

			interactionGroup.appendChild(overlayRect);
			interactionGroup.appendChild(pointerLine);

			const hideTooltip = () => {
				pointerLine.classList.remove("chart-pointer-line--active");
				chartTooltipEl.classList.add("hidden");
				chartTooltipEl.setAttribute("aria-hidden", "true");
				chartTooltipEl.style.left = "-9999px";
				chartTooltipEl.style.top = "-9999px";
			};

			const positionTooltip = (clientX, clientY) => {
				const wrapperRect = chartWrapper.getBoundingClientRect();
				const tooltipRect = chartTooltipEl.getBoundingClientRect();
				let left = clientX - wrapperRect.left + 16;
				let top = clientY - wrapperRect.top - tooltipRect.height - 16;
				const padding = 8;

				if (left + tooltipRect.width > wrapperRect.width - padding) {
					left = wrapperRect.width - tooltipRect.width - padding;
				}
				if (left < padding) {
					left = padding;
				}
				if (top < padding) {
					top = clientY - wrapperRect.top + 16;
				}
				if (top + tooltipRect.height > wrapperRect.height - padding) {
					top = wrapperRect.height - tooltipRect.height - padding;
				}

				chartTooltipEl.style.left = `${left}px`;
				chartTooltipEl.style.top = `${top}px`;
			};

			const updateTooltip = (entry, clientX, clientY) => {
				if (!entry) {
					return;
				}
				pointerLine.setAttribute("x1", entry.x);
				pointerLine.setAttribute("x2", entry.x);
				pointerLine.classList.add("chart-pointer-line--active");

				const { data } = entry;
				if (chartTooltipDateEl) {
					const formattedDate = data.date instanceof Date && !Number.isNaN(data.date.valueOf())
						? CHART_TOOLTIP_DATE_FORMATTER.format(data.date)
						: data.dateLabel || "";
					chartTooltipDateEl.textContent = formattedDate || "—";
				}
				if (chartTooltipLiquidEl) {
					chartTooltipLiquidEl.textContent = formatTooltipCurrency(data.liquid);
				}
				if (chartTooltipTotalEl) {
					chartTooltipTotalEl.textContent = formatTooltipCurrency(data.total);
				}

				chartTooltipEl.classList.remove("hidden");
				chartTooltipEl.setAttribute("aria-hidden", "false");
				positionTooltip(clientX, clientY);
			};

			const findNearestEntry = (hoverTime) => {
				let nearest = null;
				let nearestDistance = Infinity;
				for (const entry of tooltipPoints) {
					const distance = Math.abs(entry.time - hoverTime);
					if (distance < nearestDistance) {
						nearestDistance = distance;
						nearest = entry;
					}
				}
				return nearest;
			};

			const handleClientPoint = (clientX, clientY) => {
				const svgPoint = clientPointToSvgCoordinates(chartSvg, clientX, clientY);
				if (!svgPoint) {
					hideTooltip();
					return;
				}
				const clampedX = clampValue(svgPoint.x, plotLeftX, plotRightX);
				const domainSpan = Math.max(1, xMax - xMin);
				const ratio = domainSpan === 0 ? 0 : (clampedX - plotLeftX) / Math.max(1, plotWidth);
				const hoverTime = xMin + ratio * (xMax - xMin);
				const nearest = findNearestEntry(hoverTime);
				if (!nearest) {
					hideTooltip();
					return;
				}
				updateTooltip(nearest, clientX, clientY);
			};

			const handleMouseMove = (event) => {
				handleClientPoint(event.clientX, event.clientY);
			};

			const handleTouchMove = (event) => {
				if (!event.touches || event.touches.length === 0) {
					hideTooltip();
					return;
				}
				const touch = event.touches[0];
				event.preventDefault();
				handleClientPoint(touch.clientX, touch.clientY);
			};

			overlayRect.addEventListener("mouseenter", handleMouseMove);
			overlayRect.addEventListener("mousemove", handleMouseMove);
			overlayRect.addEventListener("mouseleave", hideTooltip);
			overlayRect.addEventListener("touchstart", handleTouchMove, { passive: false });
			overlayRect.addEventListener("touchmove", handleTouchMove, { passive: false });
			overlayRect.addEventListener("touchend", hideTooltip);
			overlayRect.addEventListener("touchcancel", hideTooltip);

			hideTooltip();
		}

		function formatTooltipCurrency(value) {
			if (typeof value !== "number" || !Number.isFinite(value)) {
				return "—";
			}
			return CHART_TOOLTIP_CURRENCY_FORMATTER.format(value);
		}

		function clampValue(value, min, max) {
			if (!Number.isFinite(value)) {
				return value;
			}
			if (value < min) {
				return min;
			}
			if (value > max) {
				return max;
			}
			return value;
		}

		function clientPointToSvgCoordinates(svg, clientX, clientY) {
			if (!svg || typeof svg.createSVGPoint !== "function") {
				return null;
			}
			const point = svg.createSVGPoint();
			point.x = clientX;
			point.y = clientY;
			const screenCTM = svg.getScreenCTM();
			if (!screenCTM || typeof screenCTM.inverse !== "function") {
				return null;
			}
			return point.matrixTransform(screenCTM.inverse());
		}

	function scheduleChartRerender() {
		if (!forecastDataset || !chartWrapper || !chartSvg) {
			return;
		}
		if (resultsPanel && resultsPanel.hidden) {
			return;
		}
		if (chartResizeFrame !== null) {
			return;
		}
		chartResizeFrame = window.requestAnimationFrame(() => {
			chartResizeFrame = null;
			renderScenarioChart();
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

	if (!cloned.recommendations || typeof cloned.recommendations !== "object") {
		cloned.recommendations = {};
	}
	if (cloned.recommendations.emergencyFundMonths === undefined) {
		cloned.recommendations.emergencyFundMonths = 6;
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
	clearStickyInlineError();
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
	simGrid.appendChild(createInputField({
		label: "Emergency fund target (months)",
		path: "recommendations.emergencyFundMonths",
		value: currentConfig.recommendations?.emergencyFundMonths ?? "",
		inputType: "number",
		step: "0.1",
		arrowStep: ARROW_STEP_SMALL,
		tooltip: "Months of expenses to target for the emergency fund recommendation. Set to 0 to disable.",
		validation: { type: "number", min: 0, max: 120 },
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
	queuePersistEditorState();
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

function generateDuplicateScenarioName(originalName) {
	const fallback = "Scenario";
	const trimmedOriginal = typeof originalName === "string" ? originalName.trim() : "";
	const baseName = trimmedOriginal !== "" ? trimmedOriginal : fallback;
	const existingNames = new Set(
		(Array.isArray(currentConfig?.scenarios) ? currentConfig.scenarios : []).map((scenario) =>
			typeof scenario?.name === "string" ? scenario.name.trim().toLowerCase() : "",
		),
	);
	let suffix = 1;
	let candidate;
	do {
		candidate = suffix === 1 ? `${baseName} copy` : `${baseName} copy ${suffix}`;
		suffix += 1;
	} while (existingNames.has(candidate.trim().toLowerCase()));
	return candidate;
}


function createScenarioCard(scenario, index) {
	const card = document.createElement("div");
	card.className = "editor-card";

	const duplicateScenario = () => {
		if (!Array.isArray(currentConfig?.scenarios)) {
			return;
		}
		const original = currentConfig.scenarios[index];
		if (!original) {
			return;
		}
		const clone = cloneDeep(original) || createEmptyScenario();
		clone.name = generateDuplicateScenarioName(original.name);
		currentConfig.scenarios.splice(index + 1, 0, clone);
		renderConfigEditor();
		switchTab("config");
	};

	const removeScenario = () => {
		if (!Array.isArray(currentConfig?.scenarios) || currentConfig.scenarios.length <= 1) {
			return;
		}
		currentConfig.scenarios.splice(index, 1);
		renderConfigEditor();
	};

	const canRemoveScenario = Array.isArray(currentConfig?.scenarios) && currentConfig.scenarios.length > 1;

	const { header, title } = createCardHeader(
		scenario.name || `Scenario ${index + 1}`,
		removeScenario,
		"Remove scenario",
		{
			extraClass: "scenario-card-header",
			extraActions: [
				{
					label: "Duplicate",
					onClick: duplicateScenario,
					tooltip: "Make a copy of this scenario",
					variant: "secondary",
				},
			],
			removeTooltip: "Remove scenario",
			removeDisabled: !canRemoveScenario,
			removeDisabledTooltip: "At least one scenario must remain.",
		},
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
	if (options.headingClass && options.headingClass.indexOf("sticky-heading") !== -1) {
		container.classList.add("sticky-heading-container");
	}

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

function createEventOptimizerSection(event, basePath, options = {}) {
	if (options.allowOptimizer === false) {
		return null;
	}
	if (typeof basePath !== "string" || !basePath.startsWith("scenarios[")) {
		return null;
	}

	const section = document.createElement("div");
	section.className = "editor-optimizer";
	if (!optimizerEnabled) {
		section.classList.add("editor-optimizer--global-disabled");
	}

	const header = document.createElement("div");
	header.className = "editor-optimizer__header";
	const labelEl = document.createElement("span");
	labelEl.className = "editor-optimizer__label";
	labelEl.textContent = "Optimizer";
	header.appendChild(labelEl);
	section.appendChild(header);

	const initialHasOptimizer = Boolean(event && typeof event.optimize === "object");
	const initialField = normalizeOptimizerField(initialHasOptimizer && event.optimize?.field ? event.optimize.field : "amount");
	const resolveTooltipMessage = (hasOptimizer, fieldKey) => {
		if (!optimizerEnabled) {
			return "Turn on Run optimizer in the toolbar to adjust this event automatically.";
		}
		if (!hasOptimizer) {
			return "Enable the optimizer to adjust this event while keeping cash above the emergency-fund floor.";
		}
		return getOptimizerFieldDescription(fieldKey);
	};

	const initialTooltipMessage = resolveTooltipMessage(initialHasOptimizer, initialField);
	const helpElements = attachFieldHelp({
		wrapper: section,
		labelEl,
		tooltipText: initialTooltipMessage,
		label: "Optimizer",
	});
	const updateHelpTooltip = (message) => {
		const content = message || "Optimizer details";
		if (helpElements?.tooltip) {
			helpElements.tooltip.textContent = content;
		}
		if (helpElements?.trigger) {
			helpElements.trigger.setAttribute("aria-label", "Optimizer info");
		}
	};

	const toggleLabel = document.createElement("label");
	toggleLabel.className = "editor-optimizer__toggle";
	const toggleInput = document.createElement("input");
	toggleInput.type = "checkbox";
	toggleInput.checked = initialHasOptimizer;
	toggleInput.disabled = !optimizerEnabled;
	toggleInput.setAttribute("aria-label", initialHasOptimizer ? "Disable optimizer for this event" : "Enable optimizer for this event");
	toggleLabel.appendChild(toggleInput);
	const toggleText = document.createElement("span");
	toggleText.className = "editor-optimizer__toggle-text";
	toggleText.textContent = "Enable";
	toggleLabel.appendChild(toggleText);
	header.appendChild(toggleLabel);

	const optimizerPathPrefix = `${basePath}.optimize`;

	const clearOptimizerGrid = () => {
		const existingGrid = section.querySelector(".editor-optimizer__grid");
		if (existingGrid) {
			existingGrid.remove();
		}
		removeRegisteredInputsByPrefix(`${optimizerPathPrefix}.`);
	};

	const renderOptimizerContent = () => {
		clearOptimizerGrid();

		const hasOptimizer = Boolean(event && typeof event.optimize === "object");
		toggleInput.disabled = !optimizerEnabled;
		toggleInput.checked = hasOptimizer;
		section.classList.toggle("editor-optimizer--has-config", hasOptimizer);

		const normalizedField = hasOptimizer && event.optimize ? normalizeOptimizerField(event.optimize.field) : initialField;
		const message = resolveTooltipMessage(hasOptimizer, normalizedField);
		toggleLabel.title = message || "";
		if (hasOptimizer) {
			toggleInput.setAttribute("aria-label", "Disable optimizer for this event");
		} else {
			toggleInput.setAttribute("aria-label", optimizerEnabled ? "Enable optimizer for this event" : "Optimizer disabled while Run optimizer is off");
		}
		updateHelpTooltip(message);

		if (!optimizerEnabled || !hasOptimizer) {
			return;
		}

		const optimizerConfig = ensureOptimizerDefaults(event, normalizedField);

		const optimizerGrid = document.createElement("div");
		optimizerGrid.className = "editor-grid editor-optimizer__grid";

		const fieldSelect = createInputField({
			label: "Parameter",
			path: `${optimizerPathPrefix}.field`,
			value: normalizedField,
			inputType: "select",
			options: OPTIMIZER_FIELD_OPTIONS,
			tooltip: "Choose which detail the optimizer may adjust for this event.",
			onChange: (selected) => {
				const nextField = normalizeOptimizerField(selected);
				ensureOptimizerDefaults(event, nextField, { resetBounds: true, resetTolerance: true });
				queuePersistEditorState();
				renderOptimizerContent();
			},
		});
		optimizerGrid.appendChild(fieldSelect);

		const meta = getOptimizerFieldMeta(normalizedField);
		const minValue = meta.minPath === "minDate"
			? (typeof optimizerConfig.minDate === "string" ? optimizerConfig.minDate : "")
			: (Number.isFinite(optimizerConfig.min) ? optimizerConfig.min : "");
		const maxValue = meta.maxPath === "maxDate"
			? (typeof optimizerConfig.maxDate === "string" ? optimizerConfig.maxDate : "")
			: (Number.isFinite(optimizerConfig.max) ? optimizerConfig.max : "");

		optimizerGrid.appendChild(createInputField({
			label: meta.minLabel,
			path: `${optimizerPathPrefix}.${meta.minPath}`,
			value: minValue,
			inputType: meta.inputType,
			step: meta.step,
			arrowStep: meta.arrowStep,
			numberKind: meta.numberKind,
			enableNowShortcut: Boolean(meta.enableNowShortcut),
			tooltip: meta.minTooltip,
			validation: meta.validation,
		}));

		optimizerGrid.appendChild(createInputField({
			label: meta.maxLabel,
			path: `${optimizerPathPrefix}.${meta.maxPath}`,
			value: maxValue,
			inputType: meta.inputType,
			step: meta.step,
			arrowStep: meta.arrowStep,
			numberKind: meta.numberKind,
			enableNowShortcut: Boolean(meta.enableNowShortcut),
			tooltip: meta.maxTooltip,
			validation: meta.validation,
		}));

		optimizerGrid.appendChild(createInputField({
			label: meta.toleranceLabel,
			path: `${optimizerPathPrefix}.tolerance`,
			value: Number.isFinite(optimizerConfig.tolerance) ? optimizerConfig.tolerance : "",
			inputType: meta.toleranceInputType,
			step: meta.toleranceStep,
			arrowStep: meta.toleranceArrowStep,
			numberKind: meta.toleranceNumberKind,
			tooltip: meta.toleranceTooltip,
			validation: meta.toleranceValidation,
		}));

		optimizerGrid.appendChild(createInputField({
			label: "Max iterations",
			path: `${optimizerPathPrefix}.maxIterations`,
			value: Number.isFinite(optimizerConfig.maxIterations) ? optimizerConfig.maxIterations : "",
			inputType: "number",
			step: "1",
			arrowStep: ARROW_STEP_SMALL,
			numberKind: "int",
			tooltip: "Maximum optimizer solver steps before stopping.",
			validation: { type: "integer", min: 1 },
		}));

		section.appendChild(optimizerGrid);
	};

	renderOptimizerContent();

	toggleInput.addEventListener("change", () => {
		if (!optimizerEnabled) {
			toggleInput.checked = false;
			return;
		}
		if (toggleInput.checked) {
			const nextField = normalizeOptimizerField(event.optimize && event.optimize.field ? event.optimize.field : "amount");
			ensureOptimizerDefaults(event, nextField, { resetBounds: true, resetTolerance: true });
		} else {
			delete event.optimize;
			deleteConfigAtPath(`${basePath}.optimize`);
		}
		queuePersistEditorState();
		renderOptimizerContent();
	});

	return section;
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

	const optimizerSection = createEventOptimizerSection(event, basePath, options);
	if (optimizerSection) {
		card.appendChild(optimizerSection);
	}

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
	grid.appendChild(createInputField({
		label: "Tax rate on withdrawals (%)",
		path: `${basePath}.withdrawalTaxRate`,
		value: investment.withdrawalTaxRate ?? "",
		inputType: "number",
		step: "0.01",
		arrowStep: ARROW_STEP_SMALL,
		tooltip: "Tax rate applied to the growth portion of withdrawals (e.g. gains taxed when funds are distributed).",
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
		allowOptimizer: false,
		amountTooltip: "Amount contributed each time this event occurs. Enter a positive value; contributions increase this investment's balance.",
	}));

	card.appendChild(createEventCollection(investment.withdrawals, `${basePath}.withdrawals`, {
		heading: "Withdrawals",
		titlePrefix: "Withdrawal",
		addLabel: "Add withdrawal",
		emptyMessage: "No withdrawals scheduled.",
		enableWithdrawalPercentage: true,
		allowOptimizer: false,
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
		allowOptimizer: false,
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

	const extraActions = Array.isArray(options.extraActions) ? options.extraActions : [];
	const shouldRenderActions = typeof onRemove === "function" || extraActions.length > 0;
	if (shouldRenderActions) {
		const actions = document.createElement("div");
		actions.className = "editor-inline-actions";
		extraActions.forEach((action) => {
			if (!action || typeof action.onClick !== "function") {
				return;
			}
			const button = document.createElement("button");
			button.type = "button";
			button.textContent = action.label || "Action";
			button.classList.add("editor-inline-actions__button");
			if (action.tooltip) {
				button.title = action.tooltip;
			}
			if (action.variant) {
				button.classList.add(`editor-inline-actions__button--${action.variant}`);
			}
			if (action.disabled) {
				button.disabled = true;
				if (action.disabledTooltip) {
					button.title = action.disabledTooltip;
				}
				button.setAttribute("aria-disabled", "true");
			} else {
				button.addEventListener("click", action.onClick);
			}
			actions.appendChild(button);
		});
		if (typeof onRemove === "function") {
			const removeButton = document.createElement("button");
			removeButton.type = "button";
			removeButton.textContent = removeLabel || "Remove";
			removeButton.classList.add("editor-inline-actions__button", "editor-inline-actions__button--danger");
			const removeTooltip = options.removeTooltip || removeLabel || "Remove";
			if (options.removeDisabled) {
				removeButton.disabled = true;
				removeButton.setAttribute("aria-disabled", "true");
				removeButton.title = options.removeDisabledTooltip || removeTooltip;
			} else {
				removeButton.title = removeTooltip;
				removeButton.addEventListener("click", onRemove);
			}
			actions.appendChild(removeButton);
		}
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

function removeRegisteredInputsByPrefix(prefix) {
	if (!prefix) {
		return;
	}
	registeredInputs = registeredInputs.filter((entry) => {
		return !(entry.path && entry.path.startsWith(prefix));
	});
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
		withdrawalTaxRate: 0,
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
		recommendations: {
			emergencyFundMonths: 6,
		},
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
	if (activeTab !== "config") {
		clearStickyInlineError();
	}

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

	if (tabName === "results" && forecastDataset) {
		renderScenarioChart();
	}
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
		const configPayload = buildConfigPayload();
		const requestBody = {
			config: configPayload,
			options: {
				optimize: Boolean(optimizerEnabled),
			},
		};
		const response = await fetch("/api/editor/forecast", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(requestBody),
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

	clearPersistedEditorState();
	currentConfig = createInitialConfig();
	hiddenLogging = getDefaultLoggingConfig();
	latestConfigYaml = "";
	clearResultsView();
	if (configDownloadUrl) {
		URL.revokeObjectURL(configDownloadUrl);
		configDownloadUrl = null;
	}
	setOptimizerEnabledState(false, { skipRender: true });
	setDataAvailability(false);
	renderConfigEditor();
	switchTab("config");
	showMessage("Configuration reset. Start building your new plan.", "success");
}

function initializeWorkspace() {
	let restoredState = null;
	if (!defaultConfigInitialized) {
		restoredState = loadPersistedEditorState();
		if (restoredState && restoredState.config) {
			currentConfig = restoredState.config;
			hiddenLogging = restoredState.logging || getDefaultLoggingConfig();
			latestConfigYaml = "";
		} else {
			currentConfig = createInitialConfig();
			hiddenLogging = getDefaultLoggingConfig();
			latestConfigYaml = "";
		}
		defaultConfigInitialized = true;
	}

	setupEditorPersistenceHandlers();
	clearResultsView();
	renderConfigEditor();
	setDataAvailability(false);
	switchTab("config");

	if (restoredState) {
		showMessage("Restored your in-progress plan from the previous session. Continue editing or reset if you'd like to start fresh.", "success");
	} else {
		showMessage("Start by building a plan or upload an existing YAML configuration.", null);
	}
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

	queuePersistEditorState();
}

function deleteConfigAtPath(path) {
	if (!currentConfig) {
		return;
	}

	const segments = splitPath(path);
	if (segments.length === 0) {
		return;
	}

	const { container, key } = getContainerByPath(currentConfig, segments, false);
	if (!container || key === undefined || key === null) {
		return;
	}

	if (Array.isArray(container) && typeof key === "number") {
		container.splice(key, 1);
	} else {
		delete container[key];
	}

	queuePersistEditorState();
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

async function initializeVersionFooter() {
	if (!versionFooter || !versionLabel) {
		return;
	}

	try {
		const response = await fetch("/api/version", { cache: "no-store" });
		if (!response.ok) {
			throw new Error(`unexpected status ${response.status}`);
		}
		const data = await response.json();
		const rawVersion = data && typeof data.version === "string" ? data.version.trim() : "";
		if (rawVersion !== "") {
			versionLabel.textContent = `Version ${rawVersion}`;
			versionFooter.classList.remove("hidden");
		}
	} catch (error) {
		console.warn("Unable to load application version.", error);
	}
}
