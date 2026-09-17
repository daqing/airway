Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
let _tanstack_query_core = require("@tanstack/query-core");
let preact_hooks = require("preact/hooks");
//#region src/errorBoundaryUtils.ts
const ensurePreventErrorBoundaryRetry = (options, errorResetBoundary, query) => {
	const throwOnError = query?.state.error && typeof options.throwOnError === "function" ? (0, _tanstack_query_core.shouldThrowError)(options.throwOnError, [query.state.error, query]) : options.throwOnError;
	if (options.suspense || throwOnError) {
		if (!errorResetBoundary.isReset()) options.retryOnMount = false;
	}
};
const useClearResetErrorBoundary = (errorResetBoundary) => {
	(0, preact_hooks.useEffect)(() => {
		errorResetBoundary.clearReset();
	}, [errorResetBoundary]);
};
const getHasError = ({ result, errorResetBoundary, throwOnError, query, suspense }) => {
	return result.isError && !errorResetBoundary.isReset() && !result.isFetching && query && (suspense && result.data === void 0 || (0, _tanstack_query_core.shouldThrowError)(throwOnError, [result.error, query]));
};
//#endregion
exports.ensurePreventErrorBoundaryRetry = ensurePreventErrorBoundaryRetry;
exports.getHasError = getHasError;
exports.useClearResetErrorBoundary = useClearResetErrorBoundary;

//# sourceMappingURL=errorBoundaryUtils.cjs.map