import { useMemo } from 'react';
import { Box, Typography } from '@mui/material';
import { useFragment } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import TerraformModuleVersionDocsInputCard from './TerraformModuleVersionDocsInputCard';
import { TerraformModuleVersionDocsInputsFragment_variables$key } from './__generated__/TerraformModuleVersionDocsInputsFragment_variables.graphql';

interface Props {
    fragmentRef: TerraformModuleVersionDocsInputsFragment_variables$key
}

interface VariableListProps {
    title: string;
    variables: { name: string }[];
}

function VariableList({ title, variables }: VariableListProps) {
    return (
        <>
            <Typography pb={2} variant="h5">{title}</Typography>
            {variables.length === 0 && (
                <Typography mb={4}>No {title.toLowerCase()} inputs</Typography>
            )}
            {variables.map((variable: any) => (
                <TerraformModuleVersionDocsInputCard
                    key={variable.name}
                    fragmentRef={variable}
                />
            ))}
        </>
    );
}

function TerraformModuleVersionDocsInputs({ fragmentRef }: Props) {
    const data = useFragment<TerraformModuleVersionDocsInputsFragment_variables$key>(
        graphql`
            fragment TerraformModuleVersionDocsInputsFragment_variables on TerraformModuleConfigurationDetails {
                variables {
                    required
                    name
                    ...TerraformModuleVersionDocsInputCardFragment_variable
                }
            }
        `, fragmentRef
    );

    const requiredVariables = useMemo(() =>
        data.variables.filter((variable: any) => variable.required),
        [data.variables]
    );

    const optionalVariables = useMemo(() =>
        data.variables.filter((variable: any) => !variable.required),
        [data.variables]
    );

    return (
        <Box>
            {data.variables.length === 0 && <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                <Typography color="textSecondary">This module does not contain any inputs</Typography>
            </Box>}
            {data.variables.length > 0 && <>
                <Typography color="textSecondary" variant="body1" mb={2}>
                    This module contains the following inputs:
                </Typography>
                <VariableList
                    title="Required"
                    variables={requiredVariables}
                />
                <VariableList
                    title="Optional"
                    variables={optionalVariables}
                />
            </>}
        </Box>
    );
}

export default TerraformModuleVersionDocsInputs;
