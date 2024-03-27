import {
  InformationCircleIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  XCircleIcon,
} from '@heroicons/react/20/solid';

export default function Alert({
  type,
  message,
  submessage,
  sublink,
  className,
  icon = true,
}: {
  type: 'error' | 'warning' | 'success' | 'info';
  message: string;
  submessage?: string;
  sublink?: string;
  className?: string;
  icon?: boolean;
}) {
  let iconComp = null;
  let containerColour = '';
  let textColour = '';
  switch (type) {
    case 'error':
      iconComp = <XCircleIcon className="h-5 w-5 text-red-400" aria-hidden="true" />;
      containerColour = 'border-red-400 bg-red-50';
      textColour = 'text-red-700';
      break;
    case 'warning':
      iconComp = <ExclamationTriangleIcon className="h-5 w-5 text-yellow-400" aria-hidden="true" />;
      containerColour = 'border-yellow-300 bg-yellow-50';
      textColour = 'text-yellow-700';
      break;
    case 'success':
      iconComp = <CheckCircleIcon className="h-5 w-5 text-green-400" aria-hidden="true" />;
      containerColour = 'border-green-400 bg-green-50';
      textColour = 'text-green-700';
      break;
    case 'info':
      iconComp = <InformationCircleIcon className="h-5 w-5 text-blue-400" aria-hidden="true" />;
      containerColour = 'border-blue-400 bg-blue-50';
      textColour = 'text-blue-700';
      break;
  }

  return (
    <div className={`rounded-xl border-l-4 ${containerColour} p-4 ${className}`}>
      <div className="flex">
        {icon && <div className="flex-shrink-0">{iconComp}</div>}
        <div className="ml-3 flex-1 md:flex md:justify-between">
          <p className={`text-sm ${textColour}`}>{message}</p>
          {submessage && (
            <p className="mt-3 text-sm md:ml-6 md:mt-0">
              {sublink ? (
                <a href={sublink} className={`whitespace-nowrap font-bold ${textColour}`}>
                  {submessage}
                  <span aria-hidden="true"> &rarr;</span>
                </a>
              ) : (
                <div className={`whitespace-nowrap font-bold ${textColour}`}>{submessage}</div>
              )}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
