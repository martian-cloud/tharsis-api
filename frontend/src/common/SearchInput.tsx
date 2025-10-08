import { darken, styled } from '@mui/material/styles';
import TextField, { TextFieldProps } from '@mui/material/TextField';

const SearchInput = styled((props: TextFieldProps) => (
    <TextField size="small" margin="none" autoComplete='off' inputProps={{ maxLength: 120 }} {...props} />
  ))(({ theme }) => ({
    '& .MuiOutlinedInput-root': {background: darken(theme.palette.background.default, 0.5)}
  }));

export default SearchInput;
