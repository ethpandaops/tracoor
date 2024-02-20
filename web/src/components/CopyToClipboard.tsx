import { useState } from 'react';

import { ClipboardDocumentCheckIcon } from '@heroicons/react/24/outline';
import classNames from 'classnames';

export default function CopyToClipboard(props: {
  text: string;
  inverted?: boolean;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);
  const onClick = () => {
    navigator.clipboard.writeText(props.text);
    setCopied(true);
  };
  return (
    <button
      className={classNames('align-bottom group', copied ? 'animate-fade' : '', props.className)}
    >
      <ClipboardDocumentCheckIcon
        onClick={onClick}
        className={classNames(
          'h-6 w-6 transition hover:rotate-[-4deg]',
          props.inverted
            ? 'stroke-gray-50 hover:stroke-gray-100'
            : 'stroke-amber-500 hover:stroke-amber-600',
        )}
      />
    </button>
  );
}
