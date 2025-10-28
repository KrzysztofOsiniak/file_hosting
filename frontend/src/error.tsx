import { useRouteError } from "react-router-dom"

export default function Error() {
    const error = useRouteError() as Error
    return <>{error.message}</>
}