import { ArrowLongLeftIcon, ArrowLongRightIcon } from '@heroicons/react/20/solid';
import classNames from 'classnames';

const generatePageNumbers = (
  currentPage: number,
  totalPages: number,
): { value: string; key: string }[] => {
  const pageNumbers = [];
  const pageBuffer = 2;
  let startPage = Math.max(currentPage - pageBuffer, 2);
  let endPage = Math.min(currentPage + pageBuffer, totalPages - 1);

  pageNumbers.push({ value: '1', key: '1' });

  // Adjust startPage and endPage to handle edge cases
  if (startPage - 1 === 2) {
    // If only page 2 is being skipped, include it instead of ellipsis
    startPage = 2;
  }
  if (endPage + 1 === totalPages - 1) {
    // If only one page before last is being skipped, include it
    endPage = totalPages - 1;
  }

  // Leading ellipsis
  if (startPage > 2) {
    pageNumbers.push({ value: '…', key: 'leading_ellipsis' });
  }

  // Middle pages
  for (let i = startPage; i <= endPage; i++) {
    pageNumbers.push({ value: i.toString(), key: i.toString() });
  }

  // Trailing ellipsis
  if (endPage < totalPages - 1) {
    pageNumbers.push({ value: '…', key: 'trailing_ellipsis' });
  }

  // Always include the last page
  if (totalPages > 1) {
    pageNumbers.push({ value: totalPages.toString(), key: totalPages.toString() });
  }

  return pageNumbers;
};

export default function Pagination({
  currentPage,
  totalPages,
  onPageChange,
}: {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}) {
  const pageNumbers = generatePageNumbers(currentPage, totalPages);

  return (
    <nav className="flex items-center justify-between px-4 sm:px-0">
      <div className="-mt-px flex w-0 flex-1">
        <button
          onClick={() => onPageChange(Math.max(1, currentPage - 1))}
          className={classNames(
            currentPage === 1
              ? 'cursor-not-allowed text-gray-500'
              : ' hover:text-gray-500 text-gray-700',
            'inline-flex items-center p-4 text-sm font-medium bg-white/35 rounded-xl',
          )}
          disabled={currentPage === 1}
        >
          <ArrowLongLeftIcon
            className={classNames(
              currentPage === 1 ? 'text-sky-700' : 'text-sky-500',
              'mr-3 h-5 w-5',
            )}
            aria-hidden="true"
          />
          Previous
        </button>
      </div>

      <div className="hidden md:-mt-px md:flex">
        {pageNumbers.map(({ value, key }) =>
          value === '…' ? (
            <span
              key={key}
              className="inline-flex items-center py-4 px-6 text-sm font-medium text-gray-700 cursor-not-allowed"
            >
              {value}
            </span>
          ) : value === currentPage.toString() ? (
            <a
              key={key}
              className="inline-flex items-center py-4 px-6 text-sm font-medium cursor-not-allowed bg-white/35 rounded-xl text-sky-600"
              aria-current="page"
            >
              {value}
            </a>
          ) : (
            <button
              key={key}
              onClick={() => onPageChange(Number.parseInt(value))}
              className="inline-flex items-center py-4 px-6 text-sm font-medium text-gray-700  hover:text-gray-500"
            >
              {value}
            </button>
          ),
        )}
      </div>

      <div className="-mt-px flex w-0 flex-1 justify-end">
        <button
          onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
          className={classNames(
            currentPage >= totalPages
              ? 'cursor-not-allowed text-gray-500'
              : 'hover:text-gray-900 text-gray-700',
            'inline-flex items-center p-4 text-sm font-medium bg-white/35 rounded-xl',
          )}
          disabled={currentPage >= totalPages}
        >
          Next
          <ArrowLongRightIcon
            className={classNames(
              currentPage >= totalPages ? 'text-sky-700' : 'text-sky-500',
              'ml-3 h-5 w-5',
            )}
            aria-hidden="true"
          />
        </button>
      </div>
    </nav>
  );
}
