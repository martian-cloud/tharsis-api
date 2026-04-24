import SvgIcon, { SvgIconProps } from '@mui/material/SvgIcon';

function MartianAgentIcon(props: SvgIconProps) {
    return (
        <SvgIcon {...props} titleAccess="martian-agent">
            <line x1="8" y1="7" x2="5" y2="2.5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            <circle cx="5" cy="2.5" r="1.5" fill="currentColor" />
            <line x1="16" y1="7" x2="19" y2="2.5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            <circle cx="19" cy="2.5" r="1.5" fill="currentColor" />
            <path d="M14 7H10C6.13 7 3 10.13 3 14V20C3 21.11 3.9 22 5 22H19C20.11 22 21 21.11 21 20V14C21 10.13 17.87 7 14 7M19 20H5V14C5 11.24 7.24 9 10 9H14C16.76 9 19 11.24 19 14V20Z" fill="currentColor" />
            <ellipse cx="8.5" cy="15.5" rx="2.5" ry="1.5" fill="currentColor" transform="rotate(15 8.5 15.5)" />
            <ellipse cx="15.5" cy="15.5" rx="2.5" ry="1.5" fill="currentColor" transform="rotate(-15 15.5 15.5)" />
        </SvgIcon>
    );
}

export default MartianAgentIcon;
