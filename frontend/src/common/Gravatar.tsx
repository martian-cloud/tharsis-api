import Avatar from '@mui/material/Avatar';
import { SxProps, Theme } from '@mui/material/styles';
import md5 from 'md5';
import React, { useEffect, useState } from 'react';

interface Props {
    email: string;
    width?: number;
    height?: number;
    sx?: SxProps<Theme>;
}

function TimelineEventAvatar(props: Props) {
    const { email, sx } = props;
    const [emailHash, setEmailHash] = useState(md5(email.toLowerCase()));

    useEffect(() => {
        setEmailHash(md5(email.toLowerCase()));
    }, [email]);

    return (
        <Avatar
            sx={{ ...sx, width: props.width, height: props.height }}
            alt={email}
            src={`https://secure.gravatar.com/avatar/${emailHash}?s=80&d=identicon`}
        />
    );
}

export default TimelineEventAvatar;
