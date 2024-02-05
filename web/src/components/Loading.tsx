import classNames from 'classnames';

import Logo from '@assets/logo.png';

export default function Loading({
  message = 'Loading…',
  className,
  textColor,
}: {
  message?: string;
  className?: string;
  textColor?: string;
}) {
  return (
    <div
      className={classNames(
        'flex items-center justify-center',
        textColor ?? 'text-stone-900 dark:text-stone-100',
        className,
      )}
    >
      <img src={Logo} className="object-contain w-8 h-8 pr-2 animate-pulse" />
      <h3 className="font-semibold  text-gray-600">{message}</h3>
    </div>
  );
}
