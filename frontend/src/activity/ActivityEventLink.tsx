import { styled } from '@mui/material';
import Link, { LinkProps } from '../routes/Link';

const StyledLink = styled(
    (props: LinkProps) => <Link color="primary" {...props} />
)(() => ({
    fontWeight: 500
}));

export default StyledLink;
