import SvgIcon, { SvgIconProps } from '@mui/material/SvgIcon';

function TharsisIcon(props: SvgIconProps) {
    return (
        <SvgIcon viewBox="0 0 24 24" {...props}>
            {/* Bottom block */}
            <path d="M3 16.5 L12 13 L21 16.5 L12 20 Z" fill="currentColor" opacity="0.45" />
            <path d="M3 16.5 L3 18.8 L12 22.3 L12 20 Z" fill="currentColor" opacity="0.4" />
            <path d="M21 16.5 L21 18.8 L12 22.3 L12 20 Z" fill="currentColor" opacity="0.35" />
            {/* Middle block */}
            <path d="M4.5 12 L12 8.8 L19.5 12 L12 15.2 Z" fill="currentColor" opacity="0.65" />
            <path d="M4.5 12 L4.5 14.3 L12 17.5 L12 15.2 Z" fill="currentColor" opacity="0.6" />
            <path d="M19.5 12 L19.5 14.3 L12 17.5 L12 15.2 Z" fill="currentColor" opacity="0.55" />
            {/* Top block */}
            <path d="M6 7.5 L12 4.8 L18 7.5 L12 10.2 Z" fill="currentColor" opacity="0.85" />
            <path d="M6 7.5 L6 9.8 L12 12.5 L12 10.2 Z" fill="currentColor" opacity="0.8" />
            <path d="M18 7.5 L18 9.8 L12 12.5 L12 10.2 Z" fill="currentColor" opacity="0.75" />
            {/* Capstone */}
            <path d="M8.5 3.8 L12 2 L15.5 3.8 L12 5.6 Z" fill="currentColor" />
            <path d="M8.5 3.8 L8.5 5.6 L12 7.4 L12 5.6 Z" fill="currentColor" opacity="0.9" />
            <path d="M15.5 3.8 L15.5 5.6 L12 7.4 L12 5.6 Z" fill="currentColor" opacity="0.85" />
        </SvgIcon>
    );
}

export default TharsisIcon;
