import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider, redirect } from "react-router-dom"
import Login from './login.tsx'
import Signup from './signup.tsx'
import RootHeader from './rootheader.tsx'
import Home from './home.tsx'

const router = createBrowserRouter([
    {
        element: <RootHeader/>,
        children: [
            {
                path: "",
                loader: function loader() {
                    return redirect('/home')
                },
            },
            {
                path: "home",
                element: <Home/>
            },
            {
                path: "login",
                element: <Login/>
            },
            {
                path: "signup",
                element: <Signup/>
            }
        ]
    }
])

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <RouterProvider router={router}/>
    </StrictMode>,
)
