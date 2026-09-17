export function Pagination(props: {
  page: number;
  pageCount: number;
  onPageChange: (page: number) => void;
}) {
  const { page, pageCount } = props;
  if (pageCount <= 1) return null;

  return (
    <nav class="aw-pagination" aria-label="pagination">
      <button
        type="button"
        class="aw-button aw-button-secondary aw-button-sm"
        disabled={page <= 1}
        onClick={() => props.onPageChange(page - 1)}
      >
        ← Prev
      </button>
      <span>
        Page {page} / {pageCount}
      </span>
      <button
        type="button"
        class="aw-button aw-button-secondary aw-button-sm"
        disabled={page >= pageCount}
        onClick={() => props.onPageChange(page + 1)}
      >
        Next →
      </button>
    </nav>
  );
}
