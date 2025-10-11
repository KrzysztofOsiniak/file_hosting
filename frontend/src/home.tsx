import css from './css/home.module.scss'


function Home() {

    return (
    <>
    <div className={css.wrapper}>
        <div className={css.repoShadowWrapper}>
            <div className={css.repoContainer}></div>
        </div>
        <div className={css.searchShadowWrapper}>
            <div className={css.searchContainer}></div>
        </div>
    </div>
    </>
    )
}

export default Home
