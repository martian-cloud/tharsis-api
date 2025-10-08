import { useCallback } from 'react';
import { Alert, Box, TextField, Typography } from "@mui/material";
import { FormData } from "./NewManagedIdentityAliasDialog";
import { MutationError } from "../../../common/error";
import GroupAutocomplete, { GroupOption } from "../../GroupAutocomplete";

interface Props {
    data: FormData
    onChange: (data: FormData) => void
    error?: MutationError
    groupPath: string
}

function ManagedIdentityAliasForm({ data, onChange, error, groupPath }: Props) {

    const filterGroups = useCallback(((options: readonly GroupOption[]) => {
        return options.filter((opt: GroupOption) => (!opt.fullPath.startsWith(`${groupPath}/`) && opt.fullPath !== groupPath));
    }), [groupPath]);

    const onGroupPathChange = (group: GroupOption | null) => {
        onChange({ ...data, groupPath: group ? group.fullPath : '' })
    };

    return (
        <Box display="flex" flexDirection="column">
            {error && <Alert sx={{ marginBottom: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Box>
                <TextField
                    sx={{ mb: 0.5 }}
                    fullWidth
                    autoComplete="off"
                    size="small"
                    label="Name"
                    value={data.name}
                    onChange={event => onChange({ ...data, name: event.target.value })}
                />
                <Typography variant="subtitle2">Enter a unique name for this alias</Typography>
                <GroupAutocomplete
                    sx={{ mt: 2, mb: 0.5 }}
                    placeholder="Group"
                    onSelected={onGroupPathChange}
                    filterGroups={filterGroups}
                />
                <Typography variant="subtitle2">Select the group where this alias will be created</Typography>
            </Box>
        </Box>
    );
}

export default ManagedIdentityAliasForm;
