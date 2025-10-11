import { Outlet } from "react-router";
import css from './css/rootheader.module.scss'

function RootHeader() {

    return (
    <>
        <div className={css.header}></div>
        <div className={css.bodyContainer}>
            <Outlet/>
        </div>
    </>
    )
}

export default RootHeader
