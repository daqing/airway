import { useEffect, useLayoutEffect, useRef, useState } from "preact/hooks";
//#region src/utils.ts
function useSyncExternalStore(subscribe, getSnapshot) {
	const value = getSnapshot();
	const [{ _instance }, forceUpdate] = useState({ _instance: {
		_value: value,
		_getSnapshot: getSnapshot
	} });
	useLayoutEffect(() => {
		_instance._value = value;
		_instance._getSnapshot = getSnapshot;
		if (didSnapshotChange(_instance)) forceUpdate({ _instance });
	}, [
		subscribe,
		value,
		getSnapshot
	]);
	useEffect(() => {
		if (didSnapshotChange(_instance)) forceUpdate({ _instance });
		return subscribe(() => {
			if (didSnapshotChange(_instance)) forceUpdate({ _instance });
		});
	}, [subscribe]);
	return value;
}
function didSnapshotChange(inst) {
	const latestGetSnapshot = inst._getSnapshot;
	const prevValue = inst._value;
	try {
		const nextValue = latestGetSnapshot();
		return !Object.is(prevValue, nextValue);
	} catch (_error) {
		return true;
	}
}
function useSyncExternalStoreWithSelector(subscribe, getSnapshot, selector, isEqual) {
	const selectedSnapshotRef = useRef();
	const getSelectedSnapshot = () => {
		const selected = selector(getSnapshot());
		if (selectedSnapshotRef.current === void 0 || !isEqual(selectedSnapshotRef.current, selected)) selectedSnapshotRef.current = selected;
		return selectedSnapshotRef.current;
	};
	return useSyncExternalStore(subscribe, getSelectedSnapshot);
}
//#endregion
export { useSyncExternalStore, useSyncExternalStoreWithSelector };

//# sourceMappingURL=utils.js.map