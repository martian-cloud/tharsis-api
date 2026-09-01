import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes, useParams } from 'react-router-dom';
import EditCleanupPolicy from './EditCleanupPolicy';
import NewCleanupPolicy from './NewCleanupPolicy';
import CleanupPolicyList from './CleanupPolicyList';
import { CleanupPoliciesFragment_namespace$key } from './__generated__/CleanupPoliciesFragment_namespace.graphql';

interface Props {
    fragmentRef: CleanupPoliciesFragment_namespace$key;
}

function CleanupPolicies({ fragmentRef }: Props) {
    const { kind } = useParams<{ kind: string }>();

    const data = useFragment<CleanupPoliciesFragment_namespace$key>(
        graphql`
        fragment CleanupPoliciesFragment_namespace on Namespace {
            fullPath
        }
        `, fragmentRef);

    return (
        <Routes>
            <Route index element={<CleanupPolicyList namespacePath={data.fullPath} />} />
            <Route path="new" element={<NewCleanupPolicy namespacePath={data.fullPath} />} />
            <Route path=":kind/edit" element={<EditCleanupPolicy key={kind} namespacePath={data.fullPath} />} />
        </Routes>
    );
}

export default CleanupPolicies;
