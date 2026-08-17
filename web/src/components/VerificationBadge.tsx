import { useState } from 'react';

import { Popover, PopoverButton, PopoverPanel } from '@headlessui/react';
import {
  CheckBadgeIcon,
  DocumentDuplicateIcon,
  QuestionMarkCircleIcon,
  ShieldCheckIcon,
  ShieldExclamationIcon,
} from '@heroicons/react/24/outline';
import classNames from 'classnames';

function formatTimestamp(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return `${date.toISOString().slice(0, 19).replace('T', ' ')} UTC`;
}

// The four verdicts a row can carry, weakest to strongest. 'unknown' is verified bytes whose
// agreement cannot be counted (node-local artifact, or the backing blob has aged out).
type Verdict = 'unverified' | 'unknown' | 'sole' | 'agreement';

// What population agreement is measured against. Canonical artifacts (states, blocks,
// envelopes) are byte-identical across the whole fleet, so a sole copy is worth amber.
// Traces are client-specific bytes: agreement is only expected between nodes running the
// same client build, so the only node of its client is a normal condition, not a warning.
export type AgreementScope = 'fleet' | 'client';

function verdictFor(verifiedAt?: string, agreementCount?: number): Verdict {
  if (!verifiedAt) return 'unverified';
  if (agreementCount === undefined || agreementCount < 1) return 'unknown';
  return agreementCount >= 2 ? 'agreement' : 'sole';
}

const emeraldChip =
  'bg-emerald-500/10 text-emerald-700 ring-emerald-600/30 hover:bg-emerald-500/20';
const skyChip = 'bg-sky-500/10 text-sky-700 ring-sky-600/30 hover:bg-sky-500/20';

const chipStyles: Record<Verdict, string> = {
  agreement: emeraldChip,
  sole: 'bg-amber-500/10 text-amber-700 ring-amber-600/30 hover:bg-amber-500/20',
  unknown: skyChip,
  unverified: 'bg-gray-500/10 text-gray-500 ring-gray-500/20 hover:bg-gray-500/20',
};

const chipIcons: Record<Verdict, typeof CheckBadgeIcon> = {
  agreement: CheckBadgeIcon,
  sole: ShieldExclamationIcon,
  unknown: ShieldCheckIcon,
  unverified: QuestionMarkCircleIcon,
};

function headline(
  verdict: Verdict,
  scope: AgreementScope,
  agreementCount?: number,
  hashless?: boolean,
): string {
  switch (verdict) {
    case 'agreement':
      return scope === 'client'
        ? `${agreementCount} nodes on this client agree`
        : `${agreementCount} nodes agree`;
    case 'sole':
      return scope === 'client' ? 'Only node running this client' : 'Only copy so far';
    case 'unknown':
      // A verified row with no hash predates content verification: it was fetched and
      // stored, but never hashed, so the popover must not claim hashing happened.
      return hashless ? 'Fetched from this node' : 'Verified from this node';
    case 'unverified':
      return 'Unverified';
  }
}

function explanation(verdict: Verdict, scope: AgreementScope, hashless?: boolean): string {
  switch (verdict) {
    case 'agreement':
      return scope === 'client'
        ? 'Every node running this client build served byte-identical output for this item. Different clients legitimately produce different bytes, so agreement is only counted within one client and version.'
        : 'Each of these nodes served this payload from its own client, and every copy hashed to identical bytes. One copy is stored; every row is independently verified evidence.';
    case 'sole':
      return scope === 'client'
        ? 'This output is client-specific: each client serializes it differently, so agreement is only expected between nodes running the same client and version — and this is the only node with this build. Expected, not a warning.'
        : "These bytes were read and hashed from this node, but no other node's verified payload shares this hash yet. Normal moments after first capture, or when only one node serves this artifact.";
    case 'unknown':
      return hashless
        ? 'This row predates content verification: the payload was fetched and stored from this node, but its bytes were never hashed, so agreement cannot be counted.'
        : 'These bytes were read and hashed from this node. Agreement is not counted here — the artifact is node-local, or the stored payload backing this hash has since aged out.';
    case 'unverified':
      return 'Nobody has read these bytes back from this node, so this row is a claim rather than evidence.';
  }
}

function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-[10px] font-semibold uppercase tracking-wider text-gray-400">{label}</dt>
      <dd className="text-xs text-gray-700">{children}</dd>
    </div>
  );
}

export default function VerificationBadge({
  contentHash,
  verifiedAt,
  contentMatchedAt,
  agreementCount,
  scope = 'fleet',
  className,
}: {
  contentHash?: string;
  verifiedAt?: string;
  contentMatchedAt?: string;
  agreementCount?: number;
  scope?: AgreementScope;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);
  const verdict = verdictFor(verifiedAt, agreementCount);
  const hashless = !contentHash;
  const soleButExpected = verdict === 'sole' && scope === 'client';
  const Icon = soleButExpected ? ShieldCheckIcon : chipIcons[verdict];

  // The chip is just icon + count; the words live in the title and popover. States with no
  // count to show (node-local kinds, unverified rows) are icon-only, so every table wears
  // the same compact chip.
  const chipLabel = (() => {
    switch (verdict) {
      case 'agreement':
      case 'sole':
        return `${agreementCount}`;
      case 'unknown':
      case 'unverified':
        return '';
    }
  })();

  const copyHash = () => {
    if (!contentHash || !navigator.clipboard) return;
    navigator.clipboard.writeText(contentHash).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };

  return (
    <Popover className={classNames('relative inline-flex', className)}>
      <PopoverButton
        title={headline(verdict, scope, agreementCount, hashless)}
        className={classNames(
          'inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-semibold ring-1 ring-inset transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500',
          soleButExpected ? skyChip : chipStyles[verdict],
        )}
      >
        <Icon className="h-4 w-4" aria-hidden="true" />
        {chipLabel && <span>{chipLabel}</span>}
      </PopoverButton>
      <PopoverPanel
        transition
        anchor="bottom end"
        className="z-50 w-80 rounded-lg bg-white p-4 shadow-xl ring-1 ring-gray-900/10 [--anchor-gap:6px] transition duration-100 ease-out data-[closed]:scale-95 data-[closed]:opacity-0"
      >
        <div className="flex items-center gap-2">
          <Icon
            className={classNames('h-5 w-5 shrink-0', {
              'text-emerald-600': verdict === 'agreement',
              'text-amber-600': verdict === 'sole' && !soleButExpected,
              'text-sky-600': verdict === 'unknown' || soleButExpected,
              'text-gray-400': verdict === 'unverified',
            })}
            aria-hidden="true"
          />
          <h3 className="text-sm font-bold text-gray-800">
            {headline(verdict, scope, agreementCount, hashless)}
          </h3>
        </div>
        <p className="mt-2 text-xs/5 text-gray-500">{explanation(verdict, scope, hashless)}</p>
        {(contentHash || verifiedAt || contentMatchedAt) && (
          <dl className="mt-3 flex flex-col gap-2 border-t border-gray-100 pt-3">
            {contentHash && (
              <DetailRow label="Content hash (sha256)">
                <span className="flex items-start gap-1">
                  <code className="font-mono text-[11px]/4 break-all text-gray-600">
                    {contentHash}
                  </code>
                  <button
                    type="button"
                    onClick={copyHash}
                    title="Copy content hash"
                    className="shrink-0 rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                  >
                    {copied ? (
                      <span className="px-1 text-[10px] font-semibold text-emerald-600">
                        copied
                      </span>
                    ) : (
                      <DocumentDuplicateIcon className="h-3.5 w-3.5" aria-hidden="true" />
                    )}
                  </button>
                </span>
              </DetailRow>
            )}
            {verifiedAt && (
              <DetailRow label={hashless ? 'Fetched and stored' : 'Fetched, hashed and verified'}>
                {formatTimestamp(verifiedAt)}
              </DetailRow>
            )}
            {contentMatchedAt && (
              <DetailRow label="Hash matched an existing copy">
                {formatTimestamp(contentMatchedAt)}
              </DetailRow>
            )}
          </dl>
        )}
        <p className="mt-3 border-t border-gray-100 pt-2 text-[10px]/4 text-gray-400">
          Every node is fetched and hashed independently — identical bytes are stored once, and a
          hash mismatch is recorded as a divergence.
        </p>
      </PopoverPanel>
    </Popover>
  );
}
