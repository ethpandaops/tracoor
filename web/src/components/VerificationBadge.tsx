import {
  CheckBadgeIcon,
  QuestionMarkCircleIcon,
  ShieldCheckIcon,
} from '@heroicons/react/24/outline';
import classNames from 'classnames';

function formatTimestamp(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return `${date.toISOString().slice(0, 19).replace('T', ' ')} UTC`;
}

export default function VerificationBadge({
  contentHash,
  verifiedAt,
  contentMatchedAt,
  className,
}: {
  contentHash?: string;
  verifiedAt?: string;
  contentMatchedAt?: string;
  className?: string;
}) {
  // Without verified_at nobody has read these bytes back off the node, so the row is not evidence
  // of anything - never dress it up with the hash.
  if (!verifiedAt) {
    return (
      <span
        title="Unverified: these bytes have not been read back and hashed from this node."
        className={classNames(
          'inline-flex items-center gap-1 rounded-md bg-gray-500/10 px-1.5 py-0.5 text-xs font-semibold text-gray-500 ring-1 ring-inset ring-gray-500/20 cursor-help',
          className,
        )}
      >
        <QuestionMarkCircleIcon className="h-4 w-4" aria-hidden="true" />
        unverified
      </span>
    );
  }

  const matched = Boolean(contentMatchedAt);
  const hashPrefix = contentHash ? contentHash.slice(0, 8) : undefined;

  const title = [
    `Verified ${formatTimestamp(verifiedAt)}`,
    contentMatchedAt
      ? `Content matched ${formatTimestamp(contentMatchedAt)}`
      : 'Content not yet matched against another copy',
    contentHash ? `Content hash ${contentHash}` : undefined,
  ]
    .filter(Boolean)
    .join('\n');

  return (
    <span
      title={title}
      className={classNames(
        'inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-semibold ring-1 ring-inset cursor-help',
        matched
          ? 'bg-emerald-500/10 text-emerald-700 ring-emerald-600/30'
          : 'bg-sky-500/10 text-sky-700 ring-sky-600/30',
        className,
      )}
    >
      {matched ? (
        <CheckBadgeIcon className="h-4 w-4" aria-hidden="true" />
      ) : (
        <ShieldCheckIcon className="h-4 w-4" aria-hidden="true" />
      )}
      {matched ? 'matched' : 'verified'}
      {hashPrefix && <span className="font-mono font-normal">{hashPrefix}</span>}
    </span>
  );
}
